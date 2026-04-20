package service

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/apperror"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// ImportCollectionFromFile reads a collection file, detects the format, and returns a parsed
// ImportedCollection tree ready to be persisted by the import usecase.
//
// Supported formats (auto-detected by shape, not by extension):
//   - Postman Collection v2.1 (JSON)
//   - Postman Collection v2.0 (JSON, legacy)
//   - OpenAPI 3.x (JSON or YAML)
//   - Insomnia v4 export (JSON)
func ImportCollectionFromFile(path string) (*entity.ImportedCollection, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, apperror.NewWithErrorDetail(constant.ErrImportFileOpen, errors.New("empty path"))
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrImportFileOpen, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrImportFileOpen, err)
	}
	if info.Size() == 0 {
		return nil, apperror.NewWithErrorDetail(constant.ErrImportFileEmpty, nil)
	}
	if info.Size() > int64(constant.MaxImportFileBytes) {
		return nil, apperror.NewWithErrorDetail(constant.ErrImportFileTooLarge, nil)
	}

	// Read with a guarded limit in case the reported Size() is stale.
	raw, err := io.ReadAll(io.LimitReader(f, int64(constant.MaxImportFileBytes)+1))
	if err != nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrImportFileOpen, err)
	}
	if len(raw) > constant.MaxImportFileBytes {
		return nil, apperror.NewWithErrorDetail(constant.ErrImportFileTooLarge, nil)
	}
	return parseImportContent(raw)
}

// parseImportContent dispatches based on a shape probe (JSON or YAML) and falls through
// importers in order from most-specific to most-generic.
func parseImportContent(raw []byte) (*entity.ImportedCollection, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, apperror.NewWithErrorDetail(constant.ErrImportFileEmpty, nil)
	}

	// If the payload starts with '{' it's JSON — probe shape.
	if trimmed[0] == '{' {
		kind := detectJSONFormat(trimmed)
		switch kind {
		case formatPostmanV21:
			col, err := importPostmanV21(trimmed)
			if err != nil {
				return nil, apperror.NewWithErrorDetail(constant.ErrImportParseFailed, err)
			}
			return col, nil
		case formatPostmanV20:
			col, err := importPostmanV20(trimmed)
			if err != nil {
				return nil, apperror.NewWithErrorDetail(constant.ErrImportParseFailed, err)
			}
			return col, nil
		case formatInsomniaV4:
			col, err := importInsomnia(trimmed)
			if err != nil {
				return nil, apperror.NewWithErrorDetail(constant.ErrImportParseFailed, err)
			}
			return col, nil
		case formatOpenAPI3:
			col, err := importOpenAPI(trimmed)
			if err != nil {
				return nil, apperror.NewWithErrorDetail(constant.ErrImportParseFailed, err)
			}
			return col, nil
		default:
			return nil, apperror.NewWithErrorDetail(constant.ErrImportFormatUnknown, nil)
		}
	}

	// YAML fallback — only OpenAPI 3.x is supported as YAML.
	if looksLikeOpenAPIYAML(trimmed) {
		col, err := importOpenAPI(trimmed)
		if err != nil {
			return nil, apperror.NewWithErrorDetail(constant.ErrImportParseFailed, err)
		}
		return col, nil
	}
	return nil, apperror.NewWithErrorDetail(constant.ErrImportFormatUnknown, nil)
}

type importFormatKind int

const (
	formatUnknown importFormatKind = iota
	formatPostmanV21
	formatPostmanV20
	formatOpenAPI3
	formatInsomniaV4
)

// detectJSONFormat inspects the top-level keys to classify the document.
// Cheaper than full unmarshal and keeps ambiguous files from being misrouted.
func detectJSONFormat(raw []byte) importFormatKind {
	var probe struct {
		OpenAPI      string          `json:"openapi"`
		Swagger      string          `json:"swagger"`
		Info         json.RawMessage `json:"info"`
		Item         json.RawMessage `json:"item"`
		Requests     json.RawMessage `json:"requests"`
		Folders      json.RawMessage `json:"folders"`
		Paths        json.RawMessage `json:"paths"`
		Type         string          `json:"_type"`
		ExportFormat int             `json:"__export_format"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return formatUnknown
	}
	if strings.ToLower(strings.TrimSpace(probe.Type)) == "export" && probe.ExportFormat > 0 {
		return formatInsomniaV4
	}
	if strings.HasPrefix(probe.OpenAPI, "3.") && len(probe.Paths) > 0 {
		return formatOpenAPI3
	}
	if strings.HasPrefix(probe.Swagger, "2.") {
		// Not supported — surface as unknown so we return a clear error instead of half-parse.
		return formatUnknown
	}
	if len(probe.Info) > 0 {
		// Look at info.schema to distinguish Postman versions.
		var info struct {
			Schema string `json:"schema"`
		}
		_ = json.Unmarshal(probe.Info, &info)
		schema := strings.ToLower(info.Schema)
		switch {
		case strings.Contains(schema, "v2.1"):
			return formatPostmanV21
		case strings.Contains(schema, "v2.0"):
			return formatPostmanV20
		case len(probe.Item) > 0:
			return formatPostmanV21
		}
	}
	if len(probe.Requests) > 0 || len(probe.Folders) > 0 {
		return formatPostmanV20
	}
	return formatUnknown
}

// looksLikeOpenAPIYAML does a best-effort probe on YAML input without committing to a full parse.
func looksLikeOpenAPIYAML(raw []byte) bool {
	var tree map[string]interface{}
	if err := yaml.Unmarshal(raw, &tree); err != nil {
		return false
	}
	v, ok := tree["openapi"].(string)
	if !ok {
		return false
	}
	return strings.HasPrefix(v, "3.")
}
