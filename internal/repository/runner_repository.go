package repository

import (
	"PostmanJanai/ent"
	"PostmanJanai/ent/runnerrun"
	"PostmanJanai/ent/runnerrunrequest"
	"PostmanJanai/internal/entity"
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RunnerRepository persists folder runner runs and their per-request results (Phase 8).
type RunnerRepository interface {
	StartRun(ctx context.Context, in *RunnerStartInput) (string, error)
	AppendRequest(ctx context.Context, runID string, row entity.RunnerRunRequestRow) (string, error)
	UpdateProgress(ctx context.Context, runID string, passed, failed, errored, total int) error
	FinishRun(ctx context.Context, runID string, status string, durationMs int) error

	GetSummary(ctx context.Context, runID string) (*entity.RunnerRunSummary, error)
	GetDetail(ctx context.Context, runID string) (*entity.RunnerRunDetail, error)
	ListRecent(ctx context.Context, limit int) ([]entity.RunnerRunSummary, error)
	DeleteByID(ctx context.Context, runID string) error
}

// RunnerStartInput captures the snapshot fields stored on RunnerRun creation.
type RunnerStartInput struct {
	FolderID        string
	FolderName      string
	EnvironmentID   string
	EnvironmentName string
	TotalCount      int
	Notes           string
}

type runnerRepo struct {
	client *ent.Client
}

func NewRunnerRepository(client *ent.Client) RunnerRepository {
	return &runnerRepo{client: client}
}

func parseOptionalUUID(s string) (*uuid.UUID, error) {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil, nil
	}
	u, err := uuid.Parse(t)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *runnerRepo) StartRun(ctx context.Context, in *RunnerStartInput) (string, error) {
	if in == nil {
		return "", nil
	}
	b := r.client.RunnerRun.Create().
		SetStatus("running").
		SetTotalCount(in.TotalCount).
		SetPassedCount(0).
		SetFailedCount(0).
		SetErrorCount(0).
		SetFolderName(strings.TrimSpace(in.FolderName)).
		SetEnvironmentName(strings.TrimSpace(in.EnvironmentName)).
		SetNotes(strings.TrimSpace(in.Notes))

	if fid, err := parseOptionalUUID(in.FolderID); err != nil {
		return "", err
	} else if fid != nil {
		b = b.SetFolderID(*fid)
	}
	if eid, err := parseOptionalUUID(in.EnvironmentID); err != nil {
		return "", err
	} else if eid != nil {
		b = b.SetEnvironmentID(*eid)
	}

	row, err := b.Save(ctx)
	if err != nil {
		return "", err
	}
	return row.ID.String(), nil
}

func (r *runnerRepo) AppendRequest(ctx context.Context, runID string, row entity.RunnerRunRequestRow) (string, error) {
	uid, err := uuid.Parse(strings.TrimSpace(runID))
	if err != nil {
		return "", err
	}
	b := r.client.RunnerRunRequest.Create().
		SetRunID(uid).
		SetRequestName(row.RequestName).
		SetMethod(strings.TrimSpace(row.Method)).
		SetURL(strings.TrimSpace(row.URL)).
		SetStatus(strings.TrimSpace(row.Status)).
		SetStatusCode(row.StatusCode).
		SetDurationMs(row.DurationMs).
		SetResponseSizeBytes(row.ResponseSizeBytes).
		SetErrorMessage(row.ErrorMessage).
		SetSortOrder(row.SortOrder).
		SetBodyTruncated(row.BodyTruncated)

	if row.RequestID != nil {
		if rid, err := parseOptionalUUID(*row.RequestID); err == nil && rid != nil {
			b = b.SetRequestID(*rid)
		}
	}
	if len(row.Assertions) > 0 {
		if raw, err := json.Marshal(row.Assertions); err == nil {
			b = b.SetAssertionsJSON(string(raw))
		}
	}
	if len(row.Captures) > 0 {
		if raw, err := json.Marshal(row.Captures); err == nil {
			b = b.SetCapturesJSON(string(raw))
		}
	}
	if t := strings.TrimSpace(row.RequestHeadersJSON); t != "" {
		b = b.SetRequestHeadersJSON(row.RequestHeadersJSON)
	}
	if t := strings.TrimSpace(row.ResponseHeadersJSON); t != "" {
		b = b.SetResponseHeadersJSON(row.ResponseHeadersJSON)
	}
	if row.RequestBody != "" {
		b = b.SetRequestBody(row.RequestBody)
	}
	if row.ResponseBody != "" {
		b = b.SetResponseBody(row.ResponseBody)
	}
	res, err := b.Save(ctx)
	if err != nil {
		return "", err
	}
	return res.ID.String(), nil
}

func (r *runnerRepo) UpdateProgress(ctx context.Context, runID string, passed, failed, errored, total int) error {
	uid, err := uuid.Parse(strings.TrimSpace(runID))
	if err != nil {
		return err
	}
	return r.client.RunnerRun.UpdateOneID(uid).
		SetPassedCount(passed).
		SetFailedCount(failed).
		SetErrorCount(errored).
		SetTotalCount(total).
		Exec(ctx)
}

func (r *runnerRepo) FinishRun(ctx context.Context, runID string, status string, durationMs int) error {
	uid, err := uuid.Parse(strings.TrimSpace(runID))
	if err != nil {
		return err
	}
	return r.client.RunnerRun.UpdateOneID(uid).
		SetStatus(strings.TrimSpace(status)).
		SetDurationMs(durationMs).
		SetFinishedAt(time.Now()).
		Exec(ctx)
}

func (r *runnerRepo) GetSummary(ctx context.Context, runID string) (*entity.RunnerRunSummary, error) {
	uid, err := uuid.Parse(strings.TrimSpace(runID))
	if err != nil {
		return nil, err
	}
	row, err := r.client.RunnerRun.Query().Where(runnerrun.IDEQ(uid)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return runnerRunToSummary(row), nil
}

func (r *runnerRepo) GetDetail(ctx context.Context, runID string) (*entity.RunnerRunDetail, error) {
	uid, err := uuid.Parse(strings.TrimSpace(runID))
	if err != nil {
		return nil, err
	}
	row, err := r.client.RunnerRun.Query().
		Where(runnerrun.IDEQ(uid)).
		WithRequests(func(q *ent.RunnerRunRequestQuery) {
			q.Order(ent.Asc(runnerrunrequest.FieldSortOrder), ent.Asc(runnerrunrequest.FieldCreatedAt))
		}).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	sum := runnerRunToSummary(row)
	out := &entity.RunnerRunDetail{
		RunnerRunSummary: *sum,
		Notes:            row.Notes,
	}
	for _, rq := range row.Edges.Requests {
		out.Requests = append(out.Requests, runnerRequestToRow(rq))
	}
	return out, nil
}

func (r *runnerRepo) ListRecent(ctx context.Context, limit int) ([]entity.RunnerRunSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.client.RunnerRun.Query().
		Order(ent.Desc(runnerrun.FieldStartedAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entity.RunnerRunSummary, 0, len(rows))
	for _, row := range rows {
		s := runnerRunToSummary(row)
		out = append(out, *s)
	}
	return out, nil
}

func (r *runnerRepo) DeleteByID(ctx context.Context, runID string) error {
	uid, err := uuid.Parse(strings.TrimSpace(runID))
	if err != nil {
		return err
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.RunnerRunRequest.Delete().Where(runnerrunrequest.RunIDEQ(uid)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.RunnerRun.DeleteOneID(uid).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func runnerRunToSummary(row *ent.RunnerRun) *entity.RunnerRunSummary {
	out := &entity.RunnerRunSummary{
		ID:              row.ID.String(),
		FolderName:      row.FolderName,
		EnvironmentName: row.EnvironmentName,
		Status:          row.Status,
		TotalCount:      row.TotalCount,
		PassedCount:     row.PassedCount,
		FailedCount:     row.FailedCount,
		ErrorCount:      row.ErrorCount,
		DurationMs:      row.DurationMs,
		StartedAt:       row.StartedAt,
	}
	if row.FolderID != nil {
		s := row.FolderID.String()
		out.FolderID = &s
	}
	if row.EnvironmentID != nil {
		s := row.EnvironmentID.String()
		out.EnvironmentID = &s
	}
	if row.FinishedAt != nil {
		t := *row.FinishedAt
		out.FinishedAt = &t
	}
	return out
}

func runnerRequestToRow(rq *ent.RunnerRunRequest) entity.RunnerRunRequestRow {
	out := entity.RunnerRunRequestRow{
		ID:                rq.ID.String(),
		RunID:             rq.RunID.String(),
		RequestName:       rq.RequestName,
		Method:            rq.Method,
		URL:               rq.URL,
		Status:            rq.Status,
		StatusCode:        rq.StatusCode,
		DurationMs:        rq.DurationMs,
		ResponseSizeBytes: rq.ResponseSizeBytes,
		ErrorMessage:      rq.ErrorMessage,
		BodyTruncated:     rq.BodyTruncated,
		SortOrder:         rq.SortOrder,
		CreatedAt:         rq.CreatedAt,
	}
	if rq.RequestID != nil {
		s := rq.RequestID.String()
		out.RequestID = &s
	}
	if t := strings.TrimSpace(rq.AssertionsJSON); t != "" {
		var arr []entity.AssertionResult
		if err := json.Unmarshal([]byte(t), &arr); err == nil {
			out.Assertions = arr
		}
	}
	if t := strings.TrimSpace(rq.CapturesJSON); t != "" {
		var arr []entity.CaptureResult
		if err := json.Unmarshal([]byte(t), &arr); err == nil {
			out.Captures = arr
		}
	}
	if rq.RequestHeadersJSON != nil {
		out.RequestHeadersJSON = *rq.RequestHeadersJSON
	}
	if rq.ResponseHeadersJSON != nil {
		out.ResponseHeadersJSON = *rq.ResponseHeadersJSON
	}
	if rq.RequestBody != nil {
		out.RequestBody = *rq.RequestBody
	}
	if rq.ResponseBody != nil {
		out.ResponseBody = *rq.ResponseBody
	}
	return out
}
