package service

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/apperror"
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HTTPExecutor performs real HTTP requests via net/http (Phase 1).
type HTTPExecutor struct {
	Client  *http.Client
	MaxBody int64
}

func NewHTTPExecutor() *HTTPExecutor {
	t := time.Duration(constant.HTTPClientTimeoutSeconds) * time.Second
	return &HTTPExecutor{
		Client: &http.Client{
			Timeout: t,
		},
		MaxBody: int64(constant.HTTPMaxResponseBodyBytes),
	}
}

// Execute runs one HTTP request. Validation errors return (nil, err).
// After sending, transport errors (timeout, TLS, …) return *HTTPExecuteResult with ErrorMessage set and err == nil.
func (e *HTTPExecutor) Execute(ctx context.Context, in *entity.HTTPExecuteInput) (*entity.HTTPExecuteResult, error) {
	if in == nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrInvalidURL, errors.New("nil input"))
	}
	method := strings.TrimSpace(strings.ToUpper(in.Method))
	if method == "" {
		method = http.MethodGet
	}
	raw := strings.TrimSpace(in.URL)
	if raw == "" {
		return nil, apperror.NewWithErrorDetail(constant.ErrInvalidURL, errors.New("empty url"))
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrInvalidURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, apperror.NewWithErrorDetail(constant.ErrInvalidURL, errors.New("URL must include scheme and host"))
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, apperror.NewWithErrorDetail(constant.ErrInvalidURL, errors.New("only http and https are supported"))
	}
	q := u.Query()
	for _, p := range in.QueryParams {
		k := strings.TrimSpace(p.Key)
		if k != "" {
			q.Add(k, p.Value)
		}
	}
	u.RawQuery = q.Encode()
	finalURL := u.String()

	mode := entity.BodyMode(strings.TrimSpace(in.BodyMode))
	if mode == "" {
		if strings.TrimSpace(in.Body) != "" {
			mode = entity.BodyModeRaw
		} else {
			mode = entity.BodyModeNone
		}
	}

	bodyReader, contentTypeOverride, execErr := e.buildRequestBody(mode, in)
	if execErr != nil {
		return nil, execErr
	}

	req, err := http.NewRequestWithContext(ctx, method, finalURL, bodyReader)
	if err != nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrInvalidURL, err)
	}

	for _, h := range in.Headers {
		k := strings.TrimSpace(h.Key)
		if k == "" {
			continue
		}
		if contentTypeOverride != "" && strings.EqualFold(k, "Content-Type") {
			continue
		}
		req.Header.Add(k, h.Value)
	}
	if contentTypeOverride != "" {
		req.Header.Set("Content-Type", contentTypeOverride)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", constant.AppName+"/1.0")
	}

	start := time.Now()
	resp, err := e.Client.Do(req)
	duration := time.Since(start).Milliseconds()
	out := &entity.HTTPExecuteResult{DurationMs: duration}

	if err != nil {
		out.ErrorMessage = err.Error()
		return out, nil
	}
	defer resp.Body.Close()

	out.StatusCode = resp.StatusCode
	for k, vs := range resp.Header {
		for _, v := range vs {
			out.ResponseHeaders = append(out.ResponseHeaders, entity.KeyValue{Key: k, Value: v})
		}
	}

	limit := e.MaxBody
	if limit <= 0 {
		limit = int64(constant.HTTPMaxResponseBodyBytes)
	}
	limited := io.LimitReader(resp.Body, limit+1)
	data, rerr := io.ReadAll(limited)
	if rerr != nil {
		out.ErrorMessage = rerr.Error()
		return out, nil
	}
	if int64(len(data)) > limit {
		out.ResponseBody = string(data[:limit])
		out.BodyTruncated = true
		out.ResponseSizeBytes = int64(len(out.ResponseBody))
	} else {
		out.ResponseBody = string(data)
		out.ResponseSizeBytes = int64(len(data))
	}
	return out, nil
}

func (e *HTTPExecutor) buildRequestBody(mode entity.BodyMode, in *entity.HTTPExecuteInput) (io.Reader, string, error) {
	switch mode {
	case entity.BodyModeNone:
		return nil, "", nil

	case entity.BodyModeRaw:
		if strings.TrimSpace(in.Body) == "" {
			return nil, "", nil
		}
		return bytes.NewReader([]byte(in.Body)), "", nil

	case entity.BodyModeFormURLEncoded:
		v := url.Values{}
		for _, f := range in.FormFields {
			k := strings.TrimSpace(f.Key)
			if k != "" {
				v.Add(k, f.Value)
			}
		}
		encoded := v.Encode()
		if encoded == "" {
			return nil, "application/x-www-form-urlencoded", nil
		}
		return strings.NewReader(encoded), "application/x-www-form-urlencoded", nil

	case entity.BodyModeMultipartFormData:
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		for _, p := range in.MultipartParts {
			k := strings.TrimSpace(p.Key)
			if k == "" {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(p.Kind)) {
			case "file":
				if strings.TrimSpace(p.FilePath) == "" {
					continue
				}
				f, err := os.Open(p.FilePath)
				if err != nil {
					_ = mw.Close()
					return nil, "", apperror.NewWithErrorDetail(constant.ErrInvalidURL, err)
				}
				part, err := mw.CreateFormFile(k, filepath.Base(p.FilePath))
				if err != nil {
					_ = f.Close()
					_ = mw.Close()
					return nil, "", apperror.NewWithErrorDetail(constant.ErrInvalidURL, err)
				}
				_, err = io.Copy(part, f)
				_ = f.Close()
				if err != nil {
					_ = mw.Close()
					return nil, "", apperror.NewWithErrorDetail(constant.ErrInvalidURL, err)
				}
			default:
				if err := mw.WriteField(k, p.Value); err != nil {
					_ = mw.Close()
					return nil, "", apperror.NewWithErrorDetail(constant.ErrInvalidURL, err)
				}
			}
		}
		contentType := mw.FormDataContentType()
		if err := mw.Close(); err != nil {
			return nil, "", apperror.NewWithErrorDetail(constant.ErrInvalidURL, err)
		}
		if buf.Len() == 0 {
			return nil, "", nil
		}
		return bytes.NewReader(buf.Bytes()), contentType, nil

	default:
		if strings.TrimSpace(in.Body) != "" {
			return bytes.NewReader([]byte(in.Body)), "", nil
		}
		return nil, "", nil
	}
}
