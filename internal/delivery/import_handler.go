package delivery

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/service"
	"PostmanJanai/internal/usecase"
	"context"
	"errors"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ImportHandler exposes collection import endpoints to the Wails frontend.
type ImportHandler struct {
	ctx context.Context
	uc  usecase.ImportUsecase
}

func NewImportHandler(uc usecase.ImportUsecase) *ImportHandler {
	return &ImportHandler{uc: uc}
}

func (h *ImportHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *ImportHandler) getContext() context.Context {
	if h.ctx == nil {
		return context.Background()
	}
	return h.ctx
}

// PickCollectionFile opens a native file picker limited to supported collection formats.
// Returns an empty string when the user cancels the dialog.
func (h *ImportHandler) PickCollectionFile() (string, error) {
	ctx := h.getContext()
	return runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "Select a collection file",
		Filters: []runtime.FileFilter{
			{DisplayName: "Collection files (*.json, *.yaml, *.yml)", Pattern: "*.json;*.yaml;*.yml"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
}

// PreviewCollectionFile parses the file but does NOT persist — useful for UI preview before
// committing to an import (e.g., show format, warnings, counts).
func (h *ImportHandler) PreviewCollectionFile(path string) (*entity.ImportedCollection, error) {
	ctx := h.getContext()
	logger.D().InfoContext(ctx, "ImportHandler.PreviewCollectionFile", "path", path)
	col, err := service.ImportCollectionFromFile(path)
	if err != nil {
		logger.L().InfoContext(ctx, "import preview failed", "error", err)
		return nil, err
	}
	return col, nil
}

// ImportCollectionFile parses + persists a collection file. Returns the import result summary
// so the UI can navigate to the newly-created root folder and toast counts/warnings.
func (h *ImportHandler) ImportCollectionFile(path string, opts *entity.ImportOptions) (*entity.ImportResult, error) {
	ctx := h.getContext()
	logger.D().InfoContext(ctx, "ImportHandler.ImportCollectionFile", "path", path)
	col, err := service.ImportCollectionFromFile(path)
	if err != nil {
		return nil, err
	}
	var o entity.ImportOptions
	if opts != nil {
		o = *opts
	}
	res, err := h.uc.PersistCollection(ctx, col, o)
	if err != nil {
		logger.L().ErrorContext(ctx, "import persist failed", "error", err)
		return res, err
	}
	if res == nil {
		return nil, errors.New("import produced no result")
	}
	logger.L().InfoContext(ctx, "import success",
		"root_folder", res.RootFolderName,
		"folders", res.FoldersCreated,
		"requests", res.RequestsCreated,
		"environment", res.EnvironmentName,
	)
	return res, nil
}
