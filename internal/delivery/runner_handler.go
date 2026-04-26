package delivery

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/service"
	"PostmanJanai/internal/usecase"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// RunnerHandler bridges the Phase 8 RunnerUsecase to the frontend.
//
// RunFolder is fire-and-forget from the UI's perspective: the handler kicks
// off the run on a background goroutine and emits Wails events as it makes
// progress. The synchronous return value is the run id so the UI can correlate
// later "load detail" calls.
type RunnerHandler struct {
	ctx     context.Context
	uc      usecase.RunnerUsecase
	running int32

	mu       sync.Mutex
	cancel   context.CancelFunc
	activeID string
}

func NewRunnerHandler(uc usecase.RunnerUsecase) *RunnerHandler {
	return &RunnerHandler{uc: uc}
}

func (h *RunnerHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *RunnerHandler) getContext() context.Context {
	if h.ctx == nil {
		return context.Background()
	}
	return h.ctx
}

// wailsEmitter forwards usecase events to the Wails JS bus.
type wailsEmitter struct {
	ctx context.Context
}

func (w wailsEmitter) Emit(name string, payload any) {
	if w.ctx == nil {
		return
	}
	runtime.EventsEmit(w.ctx, name, payload)
}

// RunFolder kicks off the runner asynchronously and returns ack metadata
// (run-id is unknown at this point because StartRun happens inside the goroutine;
// the UI relies on the "runner:started" event to pick it up).
func (h *RunnerHandler) RunFolder(in *entity.RunFolderInput) error {
	ctx := h.getContext()
	if in == nil {
		return errors.New("run input is required")
	}
	if !atomic.CompareAndSwapInt32(&h.running, 0, 1) {
		return errors.New("a runner job is already in progress")
	}
	logger.L().InfoContext(ctx, "RunnerHandler.RunFolder", "folder_id", in.FolderID, "env_id", in.EnvironmentID, "stop_on_fail", in.StopOnFail)

	runCtx, cancel := context.WithCancel(ctx)
	h.mu.Lock()
	h.cancel = cancel
	h.activeID = ""
	h.mu.Unlock()

	go func() {
		defer atomic.StoreInt32(&h.running, 0)
		defer cancel()
		emitter := wailsEmitter{ctx: ctx}
		if _, err := h.uc.RunFolder(runCtx, in, emitter); err != nil {
			logger.L().ErrorContext(ctx, "runner failed", "error", err)
			runtime.EventsEmit(ctx, constant.RunnerEventFinished, map[string]any{
				"phase":  "finished",
				"status": constant.RunnerStatusFailed,
				"error":  err.Error(),
			})
		}
	}()
	return nil
}

// CancelRun terminates the current runner goroutine (if any).
func (h *RunnerHandler) CancelRun() error {
	h.mu.Lock()
	c := h.cancel
	h.mu.Unlock()
	if c == nil {
		return nil
	}
	c()
	return nil
}

func (h *RunnerHandler) GetRun(id string) (*entity.RunnerRunDetail, error) {
	return h.uc.GetRun(h.getContext(), id)
}

func (h *RunnerHandler) ListRecentRuns(limit int) ([]entity.RunnerRunSummary, error) {
	return h.uc.ListRecent(h.getContext(), limit)
}

func (h *RunnerHandler) DeleteRun(id string) error {
	return h.uc.DeleteRun(h.getContext(), id)
}

// ExportRunReport opens a save dialog and writes the run detail to disk.
//
// `format` is "json" (default) or "md"/"markdown". Returns "" when the user
// cancels the dialog so the frontend can stay silent.
func (h *RunnerHandler) ExportRunReport(id string, format string) (string, error) {
	ctx := h.getContext()
	rid := strings.TrimSpace(id)
	if rid == "" {
		return "", errors.New("run id is required")
	}
	detail, err := h.uc.GetRun(ctx, rid)
	if err != nil {
		return "", err
	}
	if detail == nil {
		return "", errors.New("run not found")
	}

	fmtKey := strings.ToLower(strings.TrimSpace(format))
	if fmtKey == "" {
		fmtKey = "json"
	}

	defaultName := safeFilename(detail.FolderName) + "-runner-report"
	var data []byte
	var dlgFilters []runtime.FileFilter
	var ext string
	switch fmtKey {
	case "md", "markdown":
		ext = ".md"
		dlgFilters = []runtime.FileFilter{
			{DisplayName: "Markdown (*.md)", Pattern: "*.md"},
			{DisplayName: "All files", Pattern: "*"},
		}
		data = service.MarshalRunnerRunDetailMarkdown(detail)
	default:
		fmtKey = "json"
		ext = ".json"
		dlgFilters = []runtime.FileFilter{
			{DisplayName: "JSON (*.json)", Pattern: "*.json"},
			{DisplayName: "All files", Pattern: "*"},
		}
		raw, mErr := service.MarshalRunnerRunDetailJSON(detail)
		if mErr != nil {
			return "", mErr
		}
		data = raw
	}

	logger.D().InfoContext(ctx, "RunnerHandler.ExportRunReport", "run_id", rid, "format", fmtKey)
	path, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:           "Export runner report",
		DefaultFilename: defaultName + ext,
		Filters:         dlgFilters,
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ext) {
		path = path + ext
	}
	path = filepath.Clean(path)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// safeFilename keeps only filesystem-friendly characters; we don't drag a full
// slugifier in for this one-off use.
func safeFilename(in string) string {
	t := strings.TrimSpace(in)
	if t == "" {
		return "runner"
	}
	var b strings.Builder
	for _, r := range t {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune('-')
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "runner"
	}
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}

