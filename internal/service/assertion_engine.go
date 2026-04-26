package service

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// AssertionContext mirrors CaptureContext plus the metrics we need for size/duration assertions.
type AssertionContext struct {
	StatusCode      int
	DurationMs      int64
	ResponseSize    int64
	ResponseHeaders []entity.KeyValue
	ResponseBody    string
	BodyJSON        any
	BodyJSONErr     error
}

// AssertionContextFromCapture builds an assertion context from an already-parsed
// capture context, attaching the metrics that captures don't need.
func AssertionContextFromCapture(c *CaptureContext, durationMs, responseSize int64) *AssertionContext {
	if c == nil {
		return &AssertionContext{DurationMs: durationMs, ResponseSize: responseSize}
	}
	return &AssertionContext{
		StatusCode:      c.StatusCode,
		DurationMs:      durationMs,
		ResponseSize:    responseSize,
		ResponseHeaders: c.ResponseHeaders,
		ResponseBody:    c.ResponseBody,
		BodyJSON:        c.BodyJSON,
		BodyJSONErr:     c.BodyJSONErr,
	}
}

func (c *AssertionContext) findHeader(name string) (string, bool) {
	wanted := strings.TrimSpace(name)
	for _, h := range c.ResponseHeaders {
		if strings.EqualFold(strings.TrimSpace(h.Key), wanted) {
			return h.Value, true
		}
	}
	return "", false
}

// RunAssertionRules evaluates every enabled rule against ctx in input order.
// Like the capture engine it never returns an error — rule failures live on
// the AssertionResult so that the runner can persist a complete report even
// if a single rule misbehaves (bad JSONPath etc.).
func RunAssertionRules(ctx *AssertionContext, rules []entity.RequestAssertionRow) []entity.AssertionResult {
	if ctx == nil || len(rules) == 0 {
		return nil
	}
	out := make([]entity.AssertionResult, 0, len(rules))
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		res := entity.AssertionResult{
			Name:       strings.TrimSpace(rule.Name),
			Source:     strings.TrimSpace(rule.Source),
			Expression: rule.Expression,
			Operator:   strings.TrimSpace(rule.Operator),
			Expected:   rule.Expected,
		}
		actual, exists, err := assertionActual(ctx, rule)
		if err != nil && !isOpExistence(rule.Operator) {
			res.ErrorMessage = err.Error()
			out = append(out, res)
			continue
		}
		res.Actual = actual
		passed, perr := compareValues(actual, exists, rule.Operator, rule.Expected)
		if perr != nil {
			res.ErrorMessage = perr.Error()
			out = append(out, res)
			continue
		}
		res.Passed = passed
		out = append(out, res)
	}
	return out
}

func assertionActual(ctx *AssertionContext, rule entity.RequestAssertionRow) (string, bool, error) {
	source := strings.TrimSpace(rule.Source)
	switch source {
	case constant.AssertionSourceStatus:
		return strconv.Itoa(ctx.StatusCode), true, nil
	case constant.AssertionSourceDurationMs:
		return strconv.FormatInt(ctx.DurationMs, 10), true, nil
	case constant.AssertionSourceResponseSizeBytes:
		return strconv.FormatInt(ctx.ResponseSize, 10), true, nil
	case constant.AssertionSourceHeader:
		v, ok := ctx.findHeader(rule.Expression)
		return v, ok, nil
	case constant.AssertionSourceJSONBody:
		if ctx.BodyJSON == nil {
			if ctx.BodyJSONErr != nil {
				return "", false, errors.New("response body is not valid JSON: " + ctx.BodyJSONErr.Error())
			}
			return "", false, nil
		}
		v, err := EvalJSONPath(rule.Expression, ctx.BodyJSON)
		if err != nil {
			return "", false, err
		}
		return JSONValueToString(v), true, nil
	case constant.AssertionSourceRegexBody:
		pat := strings.TrimSpace(rule.Expression)
		if pat == "" {
			return "", false, errors.New("regex pattern is empty")
		}
		re, err := regexp.Compile(pat)
		if err != nil {
			return "", false, err
		}
		m := re.FindStringSubmatch(ctx.ResponseBody)
		if m == nil {
			return "", false, nil
		}
		if len(m) >= 2 {
			return m[1], true, nil
		}
		return m[0], true, nil
	default:
		return "", false, errors.New("unknown assertion source: " + source)
	}
}

func isOpExistence(op string) bool {
	op = strings.TrimSpace(op)
	return op == constant.AssertionOpExists || op == constant.AssertionOpNotExists
}

func compareValues(actual string, exists bool, op, expected string) (bool, error) {
	op = strings.TrimSpace(op)
	switch op {
	case constant.AssertionOpExists:
		return exists, nil
	case constant.AssertionOpNotExists:
		return !exists, nil
	case constant.AssertionOpEq:
		return actual == expected, nil
	case constant.AssertionOpNeq:
		return actual != expected, nil
	case constant.AssertionOpContains:
		return strings.Contains(actual, expected), nil
	case constant.AssertionOpNotContains:
		return !strings.Contains(actual, expected), nil
	case constant.AssertionOpRegex:
		re, err := regexp.Compile(expected)
		if err != nil {
			return false, err
		}
		return re.MatchString(actual), nil
	case constant.AssertionOpGT, constant.AssertionOpLT, constant.AssertionOpGTE, constant.AssertionOpLTE:
		a, err := strconv.ParseFloat(strings.TrimSpace(actual), 64)
		if err != nil {
			return false, errors.New("actual is not numeric: " + actual)
		}
		e, err := strconv.ParseFloat(strings.TrimSpace(expected), 64)
		if err != nil {
			return false, errors.New("expected is not numeric: " + expected)
		}
		switch op {
		case constant.AssertionOpGT:
			return a > e, nil
		case constant.AssertionOpLT:
			return a < e, nil
		case constant.AssertionOpGTE:
			return a >= e, nil
		case constant.AssertionOpLTE:
			return a <= e, nil
		}
	}
	return false, errors.New("unknown operator: " + op)
}
