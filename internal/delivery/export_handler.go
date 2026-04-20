package delivery

import (
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/usecase"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExportHandler writes collection exports to disk (Postman v2.1, …).
type ExportHandler struct {
	ctx context.Context
	uc  usecase.ExportUsecase
}

func NewExportHandler(uc usecase.ExportUsecase) *ExportHandler {
	return &ExportHandler{uc: uc}
}

func (h *ExportHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *ExportHandler) getContext() context.Context {
	if h.ctx == nil {
		return context.Background()
	}
	return h.ctx
}

// ExportPostmanV21 opens a save dialog and writes the root folder tree as a
// Postman Collection v2.1 JSON file. Returns an empty error when the user cancels.
func (h *ExportHandler) ExportPostmanV21(rootFolderID string) error {
	ctx := h.getContext()
	id := strings.TrimSpace(rootFolderID)
	if id == "" {
		return nil
	}
	path, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:           "Export Postman collection",
		DefaultFilename: "collection.postman.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "Postman Collection (*.json)", Pattern: "*.json"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
	if err != nil {
		return err
	}
	if strings.TrimSpace(path) == "" {
		return nil
	}
	logger.D().InfoContext(ctx, "ExportHandler.ExportPostmanV21", "path", path)
	data, err := h.uc.ExportPostmanV21CollectionJSON(ctx, id)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(strings.ToLower(path), ".json") {
		path = path + ".json"
	}
	path = filepath.Clean(path)
	return os.WriteFile(path, data, 0o644)
}
