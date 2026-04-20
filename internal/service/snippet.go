package service

import (
	"PostmanJanai/internal/entity"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// Snippet kinds accepted by RenderSnippet (case-insensitive).
const (
	SnippetKindCurlBash  = "curl_bash"
	SnippetKindCurlCmd   = "curl_cmd"
	SnippetKindFetchJS   = "fetch_js"
	SnippetKindAxiosJS   = "axios_js"
	SnippetKindHttpie    = "httpie"
)

// RenderSnippet builds a copy-paste command for the resolved request. `in`
// must already have env vars substituted and auth merged (same as HTTP execute).
func RenderSnippet(in *entity.HTTPExecuteInput, kind string) (string, error) {
	if in == nil {
		return "", fmt.Errorf("nil input")
	}
	finalURL, err := FinalURLForRequest(in)
	if err != nil {
		return "", err
	}
	mode := normalizedBodyMode(in)
	k := strings.ToLower(strings.TrimSpace(kind))
	switch k {
	case SnippetKindCurlBash, "curl":
		return renderCurlBash(in, finalURL, mode)
	case SnippetKindCurlCmd, "curl_windows":
		return renderCurlCmd(in, finalURL, mode)
	case SnippetKindFetchJS, "fetch":
		return renderFetchJS(in, finalURL, mode)
	case SnippetKindAxiosJS, "axios":
		return renderAxiosJS(in, finalURL, mode)
	case SnippetKindHttpie:
		return renderHttpie(in, finalURL, mode)
	default:
		return "", fmt.Errorf("unknown snippet kind: %q (try curl_bash, curl_cmd, fetch_js, axios_js, httpie)", kind)
	}
}

func normalizedBodyMode(in *entity.HTTPExecuteInput) entity.BodyMode {
	mode := entity.BodyMode(strings.TrimSpace(in.BodyMode))
	if mode == "" {
		if strings.TrimSpace(in.Body) != "" {
			return entity.BodyModeRaw
		}
		return entity.BodyModeNone
	}
	return mode
}

func bashSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func sortedHeadersForSnippet(in *entity.HTTPExecuteInput, mode entity.BodyMode) []entity.KeyValue {
	ctOverride := ""
	switch mode {
	case entity.BodyModeXML:
		ctOverride = "application/xml"
	case entity.BodyModeFormURLEncoded:
		ctOverride = "application/x-www-form-urlencoded"
	}
	var out []entity.KeyValue
	for _, h := range in.Headers {
		k := strings.TrimSpace(h.Key)
		if k == "" {
			continue
		}
		if ctOverride != "" && strings.EqualFold(k, "Content-Type") {
			continue
		}
		out = append(out, entity.KeyValue{Key: k, Value: h.Value})
	}
	if ctOverride != "" {
		out = append(out, entity.KeyValue{Key: "Content-Type", Value: ctOverride})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Key) < strings.ToLower(out[j].Key)
	})
	return out
}

func renderCurlBash(in *entity.HTTPExecuteInput, finalURL string, mode entity.BodyMode) (string, error) {
	method := strings.TrimSpace(strings.ToUpper(in.Method))
	if method == "" {
		method = "GET"
	}
	var b strings.Builder
	b.WriteString("curl")
	if method != "GET" && method != "HEAD" {
		b.WriteString(" -X ")
		b.WriteString(bashSingleQuote(method))
	}
	b.WriteString(" ")
	b.WriteString(bashSingleQuote(finalURL))

	hs := sortedHeadersForSnippet(in, mode)
	for _, h := range hs {
		b.WriteString(" \\\n  -H ")
		b.WriteString(bashSingleQuote(h.Key + ": " + h.Value))
	}

	switch mode {
	case entity.BodyModeNone:
		// no body
	case entity.BodyModeRaw, entity.BodyModeXML:
		if strings.TrimSpace(in.Body) != "" {
			b.WriteString(" \\\n  --data-raw ")
			b.WriteString(bashSingleQuote(in.Body))
		}
	case entity.BodyModeFormURLEncoded:
		v := url.Values{}
		for _, f := range in.FormFields {
			k := strings.TrimSpace(f.Key)
			if k != "" {
				v.Add(k, f.Value)
			}
		}
		enc := v.Encode()
		if enc != "" {
			b.WriteString(" \\\n  --data-raw ")
			b.WriteString(bashSingleQuote(enc))
		}
	case entity.BodyModeMultipartFormData:
		for _, p := range in.MultipartParts {
			k := strings.TrimSpace(p.Key)
			if k == "" {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(p.Kind)) {
			case "file":
				fp := strings.TrimSpace(p.FilePath)
				if fp == "" {
					continue
				}
				b.WriteString(" \\\n  -F ")
				b.WriteString(bashSingleQuote(k + "=@" + fp))
			default:
				b.WriteString(" \\\n  -F ")
				b.WriteString(bashSingleQuote(k + "=" + p.Value))
			}
		}
	default:
		if strings.TrimSpace(in.Body) != "" {
			b.WriteString(" \\\n  --data-raw ")
			b.WriteString(bashSingleQuote(in.Body))
		}
	}
	return b.String(), nil
}

func renderCurlCmd(in *entity.HTTPExecuteInput, finalURL string, mode entity.BodyMode) (string, error) {
	s, err := renderCurlBash(in, finalURL, mode)
	if err != nil {
		return "", err
	}
	// Bash line continuations use `\` — cmd.exe uses `^`.
	return strings.ReplaceAll(s, " \\\n  ", " ^\r\n  "), nil
}

func renderFetchJS(in *entity.HTTPExecuteInput, finalURL string, mode entity.BodyMode) (string, error) {
	method := strings.TrimSpace(strings.ToUpper(in.Method))
	if method == "" {
		method = "GET"
	}
	hs := sortedHeadersForSnippet(in, mode)
	hdrObj := map[string]string{}
	for _, h := range hs {
		k := strings.TrimSpace(h.Key)
		if k == "" {
			continue
		}
		hdrObj[k] = h.Value
	}
	bodyStr, includeBody := snippetBodyString(mode, in)
	var b strings.Builder
	b.WriteString(`const url = ` + jsonMarshalInline(finalURL) + `;\n`)
	b.WriteString(`const options = {\n  method: ` + jsonMarshalInline(method) + `,\n`)
	b.WriteString(`  headers: ` + jsonMarshalPretty(hdrObj) + `,\n`)
	if includeBody {
		b.WriteString(`  body: ` + jsonMarshalInline(bodyStr) + `,\n`)
	}
	b.WriteString(`};\n\n`)
	b.WriteString(`const response = await fetch(url, options);\n`)
	b.WriteString(`const text = await response.text();\n`)
	b.WriteString(`console.log(response.status, text);\n`)
	return b.String(), nil
}

func renderAxiosJS(in *entity.HTTPExecuteInput, finalURL string, mode entity.BodyMode) (string, error) {
	method := strings.TrimSpace(strings.ToUpper(in.Method))
	if method == "" {
		method = "GET"
	}
	hs := sortedHeadersForSnippet(in, mode)
	hdrObj := map[string]string{}
	for _, h := range hs {
		k := strings.TrimSpace(h.Key)
		if k == "" {
			continue
		}
		hdrObj[k] = h.Value
	}
	bodyStr, includeBody := snippetBodyString(mode, in)
	var b strings.Builder
	b.WriteString(`await axios({\n`)
	b.WriteString(`  method: ` + jsonMarshalInline(method) + `,\n`)
	b.WriteString(`  url: ` + jsonMarshalInline(finalURL) + `,\n`)
	b.WriteString(`  headers: ` + jsonMarshalPretty(hdrObj) + `,\n`)
	if includeBody {
		b.WriteString(`  data: ` + jsonMarshalInline(bodyStr) + `,\n`)
	}
	b.WriteString(`});\n`)
	return b.String(), nil
}

func snippetBodyString(mode entity.BodyMode, in *entity.HTTPExecuteInput) (string, bool) {
	switch mode {
	case entity.BodyModeNone:
		return "", false
	case entity.BodyModeRaw, entity.BodyModeXML:
		s := in.Body
		return s, strings.TrimSpace(s) != ""
	case entity.BodyModeFormURLEncoded:
		v := url.Values{}
		for _, f := range in.FormFields {
			k := strings.TrimSpace(f.Key)
			if k != "" {
				v.Add(k, f.Value)
			}
		}
		enc := v.Encode()
		return enc, enc != ""
	case entity.BodyModeMultipartFormData:
		return "[multipart body — use curl -F or FormData in the browser]", true
	default:
		s := in.Body
		return s, strings.TrimSpace(s) != ""
	}
}

func jsonMarshalInline(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `""`
	}
	return string(b)
}

func jsonMarshalPretty(v interface{}) string {
	b, err := json.MarshalIndent(v, "  ", "  ")
	if err != nil {
		return "{}"
	}
	lines := strings.Split(string(b), "\n")
	for i := range lines {
		if i > 0 {
			lines[i] = "  " + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

func renderHttpie(in *entity.HTTPExecuteInput, finalURL string, mode entity.BodyMode) (string, error) {
	method := strings.TrimSpace(strings.ToUpper(in.Method))
	if method == "" {
		method = "GET"
	}
	var parts []string
	parts = append(parts, "http", method, bashSingleQuote(finalURL))
	for _, q := range in.QueryParams {
		k := strings.TrimSpace(q.Key)
		if k == "" {
			continue
		}
		parts = append(parts, bashSingleQuote(k+"=="+q.Value))
	}
	hs := sortedHeadersForSnippet(in, mode)
	for _, h := range hs {
		k := strings.TrimSpace(h.Key)
		if k == "" {
			continue
		}
		parts = append(parts, bashSingleQuote(k+":"+h.Value))
	}
	switch mode {
	case entity.BodyModeRaw, entity.BodyModeXML:
		if strings.TrimSpace(in.Body) != "" {
			parts = append(parts, bashSingleQuote(in.Body))
		}
	case entity.BodyModeFormURLEncoded:
		for _, f := range in.FormFields {
			k := strings.TrimSpace(f.Key)
			if k == "" {
				continue
			}
			parts = append(parts, bashSingleQuote(k+"="+f.Value))
		}
	case entity.BodyModeMultipartFormData:
		parts = append(parts, "# multipart: use curl -F … or httpie --multipart")
	}
	return strings.Join(parts, " \\\n  "), nil
}

// SnippetKinds returns supported kind strings for UI / validation.
func SnippetKinds() []string {
	return []string{
		SnippetKindCurlBash,
		SnippetKindCurlCmd,
		SnippetKindFetchJS,
		SnippetKindAxiosJS,
		SnippetKindHttpie,
	}
}
