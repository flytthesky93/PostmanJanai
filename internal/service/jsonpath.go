package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// EvalJSONPath walks `value` using a Postman-style dotted JSONPath expression.
//
// Supported syntax (intentionally small — avoids a heavy third-party JSONPath dep):
//
//	$              root (optional; "foo.bar" works the same as "$.foo.bar")
//	.key           map access (key cannot contain dots / brackets)
//	["key"]        map access for keys that contain special characters
//	['key']        same, single-quoted form
//	[N]            integer array index (negative = from the end)
//	[*]            "all elements" — returns []any (callers may stringify if needed)
//
// Returned `value` is whatever Go primitive / map / slice the JSON decoder produced;
// callers that need a string usually wrap with `JSONValueToString`.
//
// Returns an error when the expression is malformed or any segment fails to resolve.
func EvalJSONPath(expr string, value any) (any, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" || expr == "$" {
		return value, nil
	}
	if strings.HasPrefix(expr, "$") {
		expr = expr[1:]
	}

	cur := value
	i := 0
	for i < len(expr) {
		ch := expr[i]
		switch ch {
		case '.':
			i++
			start := i
			for i < len(expr) {
				c := expr[i]
				if c == '.' || c == '[' {
					break
				}
				i++
			}
			key := expr[start:i]
			if key == "" {
				return nil, fmt.Errorf("jsonpath: empty key after '.'")
			}
			next, err := stepKey(cur, key)
			if err != nil {
				return nil, err
			}
			cur = next
		case '[':
			end := strings.IndexByte(expr[i:], ']')
			if end < 0 {
				return nil, errors.New("jsonpath: unmatched [")
			}
			inner := strings.TrimSpace(expr[i+1 : i+end])
			i += end + 1
			next, err := stepBracket(cur, inner)
			if err != nil {
				return nil, err
			}
			cur = next
		default:
			start := i
			for i < len(expr) {
				c := expr[i]
				if c == '.' || c == '[' {
					break
				}
				i++
			}
			key := expr[start:i]
			next, err := stepKey(cur, key)
			if err != nil {
				return nil, err
			}
			cur = next
		}
	}
	return cur, nil
}

func stepKey(value any, key string) (any, error) {
	if key == "" {
		return nil, errors.New("jsonpath: empty key")
	}
	switch v := value.(type) {
	case map[string]any:
		raw, ok := v[key]
		if !ok {
			return nil, fmt.Errorf("jsonpath: key %q not found", key)
		}
		return raw, nil
	case []any:
		// Support .length only as a friendly hint.
		if key == "length" {
			return float64(len(v)), nil
		}
		return nil, fmt.Errorf("jsonpath: cannot read key %q from array", key)
	default:
		return nil, fmt.Errorf("jsonpath: cannot read key %q from %T", key, value)
	}
}

func stepBracket(value any, inner string) (any, error) {
	if inner == "" {
		return nil, errors.New("jsonpath: empty []")
	}
	if inner == "*" {
		switch v := value.(type) {
		case []any:
			return v, nil
		case map[string]any:
			out := make([]any, 0, len(v))
			for _, raw := range v {
				out = append(out, raw)
			}
			return out, nil
		default:
			return nil, fmt.Errorf("jsonpath: [*] requires array or object, got %T", value)
		}
	}
	if (strings.HasPrefix(inner, "\"") && strings.HasSuffix(inner, "\"")) ||
		(strings.HasPrefix(inner, "'") && strings.HasSuffix(inner, "'")) {
		key := inner[1 : len(inner)-1]
		return stepKey(value, key)
	}
	// Numeric index — allow negatives.
	if isAllDigits(inner) || (len(inner) > 1 && inner[0] == '-' && isAllDigits(inner[1:])) {
		idx, err := strconv.Atoi(inner)
		if err != nil {
			return nil, fmt.Errorf("jsonpath: invalid index %q", inner)
		}
		arr, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("jsonpath: index applied to non-array %T", value)
		}
		if idx < 0 {
			idx = len(arr) + idx
		}
		if idx < 0 || idx >= len(arr) {
			return nil, fmt.Errorf("jsonpath: index %d out of range (len=%d)", idx, len(arr))
		}
		return arr[idx], nil
	}
	// Fallback: treat as bare key (Postman often allows [foo] meaning foo).
	return stepKey(value, inner)
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// JSONValueToString converts a primitive / map / slice into a stable string form
// suitable for assertion comparisons and capture target storage.
func JSONValueToString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		// Preserve integer formatting when possible (most JSON numbers come back as float64).
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			parts = append(parts, JSONValueToString(item))
		}
		return "[" + strings.Join(parts, ",") + "]"
	case map[string]any:
		// Stable order isn't required for our use case (assertion comparison strings),
		// but we still sort keys for determinism in tests / logs.
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		stableSort(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%q:%s", k, JSONValueToString(t[k])))
		}
		return "{" + strings.Join(parts, ",") + "}"
	default:
		return fmt.Sprintf("%v", t)
	}
}

func stableSort(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j-1] > ss[j]; j-- {
			ss[j-1], ss[j] = ss[j], ss[j-1]
		}
	}
}
