package delivery

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/repository"
	"PostmanJanai/internal/service"
	"context"
	"errors"
)

// SnippetHandler generates copy-paste snippets (curl, fetch, …) from the same
// resolved payload as HTTP execute (env vars + auth merged).
type SnippetHandler struct {
	ctx context.Context
	env repository.EnvironmentRepository
}

func NewSnippetHandler(env repository.EnvironmentRepository) *SnippetHandler {
	return &SnippetHandler{env: env}
}

func (h *SnippetHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *SnippetHandler) getContext() context.Context {
	if h.ctx == nil {
		return context.Background()
	}
	return h.ctx
}

// RenderSnippet builds a snippet string. `kind` is one of service.SnippetKind*
// (e.g. curl_bash, fetch_js). Input is substituted with active env vars and
// auth merged into headers/query like Execute.
func (h *SnippetHandler) RenderSnippet(in *entity.HTTPExecuteInput, kind string) (string, error) {
	ctx := h.getContext()
	if in == nil {
		return "", errors.New("nil input")
	}
	logger.D().InfoContext(ctx, "SnippetHandler.RenderSnippet", "kind", kind)

	vars := map[string]string{}
	if h.env != nil {
		m, err := h.env.ActiveVariableMap(ctx)
		if err != nil {
			logger.L().InfoContext(ctx, "active environment variables unavailable for snippet", "error", err)
		} else if m != nil {
			vars = m
		}
	}
	resolved := service.CloneSubstituteHTTPExecuteInput(in, vars)
	if resolved == nil {
		return "", errors.New("nil resolved input")
	}
	service.MergeAuthIntoHeadersAndQuery(resolved)

	secrets := []string{}
	if h.env != nil {
		if s, err := h.env.ActiveSecretPlaintexts(ctx); err == nil && s != nil {
			secrets = s
		}
	}
	snipIn := resolved
	if len(secrets) > 0 {
		snipIn = service.RedactHTTPExecuteInput(resolved, secrets)
	}
	return service.RenderSnippet(snipIn, kind)
}

// ListSnippetKinds returns supported kind identifiers for the UI dropdown.
func (h *SnippetHandler) ListSnippetKinds() []string {
	return service.SnippetKinds()
}
