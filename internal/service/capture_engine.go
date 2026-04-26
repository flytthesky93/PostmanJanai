package service

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// CaptureContext is the immutable per-response input handed to RunCaptureRules.
//
// `BodyJSON` may be nil if the response body does not parse as JSON; rules
// that require json_body will short-circuit with a friendly error in that case.
type CaptureContext struct {
	StatusCode      int
	ResponseHeaders []entity.KeyValue
	ResponseBody    string
	BodyJSON        any
	BodyJSONErr     error
}

// NewCaptureContext eagerly parses BodyJSON so that successive rules don't re-parse.
// Header keys keep their original casing — the engine handles case-insensitive lookups.
func NewCaptureContext(statusCode int, headers []entity.KeyValue, body string) *CaptureContext {
	ctx := &CaptureContext{
		StatusCode:      statusCode,
		ResponseHeaders: headers,
		ResponseBody:    body,
	}
	trimmed := strings.TrimSpace(body)
	if trimmed != "" {
		var v any
		if err := json.Unmarshal([]byte(trimmed), &v); err == nil {
			ctx.BodyJSON = v
		} else {
			ctx.BodyJSONErr = err
		}
	}
	return ctx
}

// FindHeader returns the first response header value matching `name` (case-insensitive),
// plus a boolean indicating whether it was found.
func (c *CaptureContext) FindHeader(name string) (string, bool) {
	if c == nil {
		return "", false
	}
	wanted := strings.TrimSpace(name)
	for _, h := range c.ResponseHeaders {
		if strings.EqualFold(strings.TrimSpace(h.Key), wanted) {
			return h.Value, true
		}
	}
	return "", false
}

// RunCaptureRules executes every enabled capture rule against ctx and returns the
// outcomes in input order. The function never returns an error — a rule failure is
// surfaced via CaptureResult.ErrorMessage so the runner can persist partial state.
func RunCaptureRules(ctx *CaptureContext, rules []entity.RequestCaptureRow) []entity.CaptureResult {
	if ctx == nil || len(rules) == 0 {
		return nil
	}
	out := make([]entity.CaptureResult, 0, len(rules))
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		res := entity.CaptureResult{
			Name:           strings.TrimSpace(rule.Name),
			Source:         strings.TrimSpace(rule.Source),
			Expression:     rule.Expression,
			TargetScope:    strings.TrimSpace(rule.TargetScope),
			TargetVariable: strings.TrimSpace(rule.TargetVariable),
		}
		val, err := captureOne(ctx, rule)
		if err != nil {
			res.ErrorMessage = err.Error()
			out = append(out, res)
			continue
		}
		res.Value = val
		res.Captured = true
		out = append(out, res)
	}
	return out
}

func captureOne(ctx *CaptureContext, rule entity.RequestCaptureRow) (string, error) {
	source := strings.TrimSpace(rule.Source)
	switch source {
	case constant.CaptureSourceStatus:
		return strconv.Itoa(ctx.StatusCode), nil
	case constant.CaptureSourceHeader:
		v, ok := ctx.FindHeader(rule.Expression)
		if !ok {
			return "", &captureErr{msg: "header not found"}
		}
		return v, nil
	case constant.CaptureSourceJSONBody:
		if ctx.BodyJSON == nil {
			if ctx.BodyJSONErr != nil {
				return "", &captureErr{msg: "response body is not valid JSON: " + ctx.BodyJSONErr.Error()}
			}
			return "", &captureErr{msg: "response body is empty"}
		}
		v, err := EvalJSONPath(rule.Expression, ctx.BodyJSON)
		if err != nil {
			return "", err
		}
		return JSONValueToString(v), nil
	case constant.CaptureSourceRegexBody:
		pat := strings.TrimSpace(rule.Expression)
		if pat == "" {
			return "", &captureErr{msg: "regex pattern is empty"}
		}
		re, err := regexp.Compile(pat)
		if err != nil {
			return "", err
		}
		m := re.FindStringSubmatch(ctx.ResponseBody)
		if m == nil {
			return "", &captureErr{msg: "regex did not match"}
		}
		if len(m) >= 2 {
			return m[1], nil
		}
		return m[0], nil
	default:
		return "", &captureErr{msg: "unknown capture source: " + source}
	}
}

type captureErr struct{ msg string }

func (e *captureErr) Error() string { return e.msg }
