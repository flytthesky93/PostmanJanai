package usecase

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/apperror"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/repository"
	"PostmanJanai/internal/service"
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RunnerEmitter is what the usecase needs to push progress events to the UI.
// The runtime delivery wires this to wails.EventsEmit; tests use a noop / capture impl.
type RunnerEmitter interface {
	Emit(eventName string, payload any)
}

type noopEmitter struct{}

func (noopEmitter) Emit(string, any) {}

// RunnerHTTPExecutor is the slice of HTTPExecutor used by the runner. Defined as
// an interface so tests can swap in a stub without standing up a real HTTP server
// per request.
type RunnerHTTPExecutor interface {
	Execute(ctx context.Context, in *entity.HTTPExecuteInput) (*entity.HTTPExecuteResult, error)
}

// RunnerUsecase orchestrates a folder run (Phase 8).
type RunnerUsecase interface {
	RunFolder(ctx context.Context, in *entity.RunFolderInput, emitter RunnerEmitter) (*entity.RunnerRunDetail, error)
	GetRun(ctx context.Context, runID string) (*entity.RunnerRunDetail, error)
	ListRecent(ctx context.Context, limit int) ([]entity.RunnerRunSummary, error)
	DeleteRun(ctx context.Context, runID string) error
}

type runnerUsecaseImpl struct {
	folders    repository.FolderRepository
	requests   repository.RequestRepository
	rules      repository.RequestRuleRepository
	envRepo    repository.EnvironmentRepository
	runs       repository.RunnerRepository
	executor   RunnerHTTPExecutor
}

func NewRunnerUsecase(
	folders repository.FolderRepository,
	requests repository.RequestRepository,
	rules repository.RequestRuleRepository,
	envRepo repository.EnvironmentRepository,
	runs repository.RunnerRepository,
	executor RunnerHTTPExecutor,
) RunnerUsecase {
	return &runnerUsecaseImpl{
		folders:  folders,
		requests: requests,
		rules:    rules,
		envRepo:  envRepo,
		runs:     runs,
		executor: executor,
	}
}

func (u *runnerUsecaseImpl) GetRun(ctx context.Context, runID string) (*entity.RunnerRunDetail, error) {
	if strings.TrimSpace(runID) == "" {
		return nil, apperror.NewWithErrorDetail(constant.ErrRunnerNotFound, nil)
	}
	return u.runs.GetDetail(ctx, runID)
}

func (u *runnerUsecaseImpl) ListRecent(ctx context.Context, limit int) ([]entity.RunnerRunSummary, error) {
	return u.runs.ListRecent(ctx, limit)
}

func (u *runnerUsecaseImpl) DeleteRun(ctx context.Context, runID string) error {
	return u.runs.DeleteByID(ctx, runID)
}

// RunFolder is the synchronous folder runner. The Wails handler is expected to
// invoke this from a goroutine; the function blocks until the run is over.
func (u *runnerUsecaseImpl) RunFolder(ctx context.Context, in *entity.RunFolderInput, emitter RunnerEmitter) (*entity.RunnerRunDetail, error) {
	if in == nil || strings.TrimSpace(in.FolderID) == "" {
		return nil, apperror.NewWithErrorDetail(constant.ErrFolderNotFound, nil)
	}
	if emitter == nil {
		emitter = noopEmitter{}
	}
	folder, err := u.folders.GetByID(ctx, in.FolderID)
	if err != nil {
		return nil, err
	}

	envName := ""
	envID := strings.TrimSpace(in.EnvironmentID)
	if envID != "" {
		// We don't have a GetByID on environments without variables; lookup via active list/sum is overkill.
		// For now just store the supplied ID; the friendly name is filled when present.
	}
	if active, err := u.envRepo.GetActiveSummary(ctx); err == nil && active != nil {
		// If caller didn't supply one explicitly, default to the active env so captures land somewhere.
		if envID == "" {
			envID = active.ID
		}
		if envID == active.ID {
			envName = active.Name
		}
	}

	plan, err := u.collectRequests(ctx, in.FolderID)
	if err != nil {
		return nil, err
	}
	if len(plan) == 0 {
		return nil, apperror.NewWithErrorDetail(constant.ErrRunnerEmpty, nil)
	}

	runID, err := u.runs.StartRun(ctx, &repository.RunnerStartInput{
		FolderID:        in.FolderID,
		FolderName:      folder.Name,
		EnvironmentID:   envID,
		EnvironmentName: envName,
		TotalCount:      len(plan),
		Notes:           strings.TrimSpace(in.Notes),
	})
	if err != nil {
		return nil, err
	}

	emitter.Emit(constant.RunnerEventStarted, map[string]any{
		"run_id":      runID,
		"total_count": len(plan),
		"folder_name": folder.Name,
	})

	memoryBag := map[string]string{}
	envBag, err := u.envRepo.ActiveVariableMap(ctx)
	if err != nil || envBag == nil {
		envBag = map[string]string{}
	}

	startedAt := time.Now()
	passed, failed, errored := 0, 0, 0
	finalStatus := constant.RunnerStatusCompleted

	for idx, item := range plan {
		if ctx.Err() != nil {
			finalStatus = constant.RunnerStatusCancelled
			break
		}
		row := u.executeOne(ctx, item, idx, envBag, memoryBag)
		if _, err := u.runs.AppendRequest(ctx, runID, row); err != nil {
			logger.L().InfoContext(ctx, "runner append failed", "error", err)
		}
		switch row.Status {
		case constant.RunnerRequestStatusPassed:
			passed++
		case constant.RunnerRequestStatusFailed:
			failed++
		case constant.RunnerRequestStatusErrored:
			errored++
		}
		_ = u.runs.UpdateProgress(ctx, runID, passed, failed, errored, len(plan))
		emitter.Emit(constant.RunnerEventRequestDone, entity.RunnerProgressEvent{
			RunID:       runID,
			TotalCount:  len(plan),
			CurrentIdx:  idx + 1,
			PassedCount: passed,
			FailedCount: failed,
			ErrorCount:  errored,
			Phase:       "request",
			Request:     &row,
		})
		if in.StopOnFail && (row.Status == constant.RunnerRequestStatusFailed || row.Status == constant.RunnerRequestStatusErrored) {
			finalStatus = constant.RunnerStatusFailed
			break
		}
	}

	if finalStatus == constant.RunnerStatusCompleted && (failed > 0 || errored > 0) {
		// Still mark completed — Postman runner reports both in summary, mirroring CI semantics.
		finalStatus = constant.RunnerStatusCompleted
	}

	durationMs := int(time.Since(startedAt).Milliseconds())
	if err := u.runs.FinishRun(ctx, runID, finalStatus, durationMs); err != nil {
		logger.L().InfoContext(ctx, "runner finish failed", "error", err)
	}

	detail, err := u.runs.GetDetail(ctx, runID)
	if err != nil {
		return nil, err
	}
	emitter.Emit(constant.RunnerEventFinished, entity.RunnerProgressEvent{
		RunID:       runID,
		TotalCount:  detail.TotalCount,
		CurrentIdx:  detail.TotalCount,
		PassedCount: detail.PassedCount,
		FailedCount: detail.FailedCount,
		ErrorCount:  detail.ErrorCount,
		Phase:       "finished",
		Status:      detail.Status,
	})
	return detail, nil
}

type runnerPlanItem struct {
	saved        *entity.SavedRequestFull
	rootFolderID string
	sortHint     int
}

// collectRequests walks the folder subtree top-down (folder.sort_order, then request.UpdatedAt desc → asc)
// and returns the run plan. Stable ordering keeps recent run reports diff-friendly.
func (u *runnerUsecaseImpl) collectRequests(ctx context.Context, folderID string) ([]runnerPlanItem, error) {
	uid, err := uuid.Parse(strings.TrimSpace(folderID))
	if err != nil {
		return nil, errors.New("invalid folder id")
	}
	rootID, err := u.folders.ResolveRootID(ctx, uid.String())
	if err != nil {
		rootID = uid.String()
	}
	var plan []runnerPlanItem
	queue := []string{folderID}
	visited := map[string]bool{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		summaries, err := u.requests.ListByFolder(ctx, current)
		if err != nil {
			return nil, err
		}
		// Sort by name (alphabetical) inside each folder for deterministic runs.
		sort.Slice(summaries, func(i, j int) bool {
			return strings.ToLower(summaries[i].Name) < strings.ToLower(summaries[j].Name)
		})
		for _, s := range summaries {
			full, err := u.requests.GetByID(ctx, s.ID)
			if err != nil {
				continue
			}
			plan = append(plan, runnerPlanItem{saved: full, rootFolderID: rootID})
		}
		children, err := u.folders.ListChildren(ctx, current)
		if err != nil {
			return nil, err
		}
		for _, c := range children {
			queue = append(queue, c.ID)
		}
	}
	return plan, nil
}

// executeOne runs one request through env+memory substitution → executor → captures → assertions.
// The function is deliberately tolerant: any error becomes a row with status=errored so the runner
// can keep going (unless the caller requested StopOnFail).
func (u *runnerUsecaseImpl) executeOne(
	ctx context.Context,
	item runnerPlanItem,
	idx int,
	envBag, memoryBag map[string]string,
) entity.RunnerRunRequestRow {
	row := entity.RunnerRunRequestRow{
		RequestName: item.saved.Name,
		Method:      strings.ToUpper(strings.TrimSpace(item.saved.Method)),
		URL:         item.saved.URL,
		Status:      constant.RunnerRequestStatusPassed,
		SortOrder:   idx,
	}
	if id := strings.TrimSpace(item.saved.ID); id != "" {
		row.RequestID = &id
	}

	root := item.rootFolderID
	in := service.SavedRequestToHTTPInput(item.saved, &root)
	if in == nil {
		row.Status = constant.RunnerRequestStatusErrored
		row.ErrorMessage = "could not build request input"
		return row
	}

	mergedVars := mergeVarBags(envBag, memoryBag)
	resolved := service.CloneSubstituteHTTPExecuteInput(in, mergedVars)
	service.MergeAuthIntoHeadersAndQuery(resolved)

	res, err := u.executor.Execute(ctx, resolved)
	if err != nil {
		row.Status = constant.RunnerRequestStatusErrored
		row.ErrorMessage = err.Error()
		// Persist whatever snapshot we already have so the user can still see
		// the resolved request even when the network leg failed.
		applyHTTPSnapshotsToRow(&row, res, resolved)
		return row
	}
	row.URL = res.FinalURL
	row.StatusCode = res.StatusCode
	row.DurationMs = int(res.DurationMs)
	row.ResponseSizeBytes = int(res.ResponseSizeBytes)
	applyHTTPSnapshotsToRow(&row, res, resolved)
	if strings.TrimSpace(res.ErrorMessage) != "" {
		row.Status = constant.RunnerRequestStatusErrored
		row.ErrorMessage = res.ErrorMessage
		return row
	}

	captureRules, _ := u.rules.ListCaptures(ctx, item.saved.ID)
	assertionRules, _ := u.rules.ListAssertions(ctx, item.saved.ID)

	capCtx := service.NewCaptureContext(res.StatusCode, res.ResponseHeaders, res.ResponseBody)
	if len(captureRules) > 0 {
		captures := service.RunCaptureRules(capCtx, captureRules)
		for i := range captures {
			c := &captures[i]
			if !c.Captured {
				continue
			}
			scope := strings.TrimSpace(c.TargetScope)
			if scope == "" {
				scope = constant.CaptureScopeEnvironment
			}
			switch scope {
			case constant.CaptureScopeEnvironment:
				if u.envRepo == nil {
					c.ErrorMessage = "no environment repository"
					continue
				}
				ok, err := u.envRepo.UpsertActiveVariable(ctx, c.TargetVariable, c.Value)
				if err != nil {
					c.ErrorMessage = err.Error()
					continue
				}
				if !ok {
					c.ErrorMessage = "no active environment"
					continue
				}
				envBag[c.TargetVariable] = c.Value
			case constant.CaptureScopeMemory:
				memoryBag[c.TargetVariable] = c.Value
			}
		}
		row.Captures = captures
	}

	if len(assertionRules) > 0 {
		assertCtx := service.AssertionContextFromCapture(capCtx, res.DurationMs, res.ResponseSizeBytes)
		row.Assertions = service.RunAssertionRules(assertCtx, assertionRules)
		anyFail := false
		for _, a := range row.Assertions {
			if !a.Passed {
				anyFail = true
				break
			}
		}
		if anyFail {
			row.Status = constant.RunnerRequestStatusFailed
		}
	}
	return row
}

// applyHTTPSnapshotsToRow copies the resolved request snapshot and the
// response payload into the runner row. The runner persists raw values
// (post-substitution) so reviewers see exactly what hit the wire — `{{var}}`
// tokens are not preserved on purpose.
//
// `res` may be nil when the executor failed before returning a result; the
// fallback path uses `resolved` to reconstruct the request snapshot via the
// shared history-snapshot helper so the user still gets the URL/headers/body.
func applyHTTPSnapshotsToRow(row *entity.RunnerRunRequestRow, res *entity.HTTPExecuteResult, resolved *entity.HTTPExecuteInput) {
	if row == nil {
		return
	}
	var reqHdrs []entity.KeyValue
	var reqBody string
	var respHdrs []entity.KeyValue
	var respBody string
	bodyTruncated := false

	if res != nil {
		reqHdrs = res.RequestHeadersSnapshot
		reqBody = res.RequestBodySnapshot
		respHdrs = res.ResponseHeaders
		respBody = res.ResponseBody
		bodyTruncated = res.BodyTruncated
	}
	if (len(reqHdrs) == 0 || reqBody == "") && resolved != nil {
		if _, hdrs, body, err := service.HTTPRequestSnapshotsForHistory(context.Background(), resolved); err == nil {
			if len(reqHdrs) == 0 {
				reqHdrs = hdrs
			}
			if reqBody == "" {
				reqBody = body
			}
		}
	}
	if len(reqHdrs) > 0 {
		if b, err := json.Marshal(reqHdrs); err == nil {
			row.RequestHeadersJSON = string(b)
		}
	}
	if reqBody != "" {
		row.RequestBody = reqBody
	}
	if len(respHdrs) > 0 {
		if b, err := json.Marshal(respHdrs); err == nil {
			row.ResponseHeadersJSON = string(b)
		}
	}
	if respBody != "" {
		row.ResponseBody = respBody
	}
	row.BodyTruncated = bodyTruncated
}

func mergeVarBags(envBag, memoryBag map[string]string) map[string]string {
	out := make(map[string]string, len(envBag)+len(memoryBag))
	for k, v := range envBag {
		out[k] = v
	}
	// Memory bag wins so a capture taken earlier in the run overrides a stale env value.
	for k, v := range memoryBag {
		out[k] = v
	}
	return out
}
