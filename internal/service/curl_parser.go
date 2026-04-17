package service

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/apperror"
	"encoding/base64"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-shellwords"
)

// ParseCurlCommand parses a shell-style cURL command into HTTPExecuteInput.
// Supported: URL, -X/--request, -H, -d/--data/--data-raw/--data-binary, --data-urlencode,
// -F/--form (multipart), --url, -G/--get (append -d to query string), -b, -A, -u (Basic auth),
// -d @file, -F name=@file. Flags like -s, -L, -k, -v are ignored.
func ParseCurlCommand(cmd string) (*entity.HTTPExecuteInput, error) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil, apperror.NewWithErrorDetail(constant.ErrInvalidCurl, errors.New("empty command"))
	}
	args, err := shellwords.Parse(cmd)
	if err != nil {
		return nil, apperror.NewWithErrorDetail(constant.ErrInvalidCurl, err)
	}
	if len(args) == 0 {
		return nil, apperror.NewWithErrorDetail(constant.ErrInvalidCurl, errors.New("no tokens"))
	}
	if base := strings.ToLower(filepath.Base(args[0])); base == "curl" || base == "curl.exe" {
		args = args[1:]
	}
	if len(args) == 0 {
		return nil, apperror.NewWithErrorDetail(constant.ErrInvalidCurl, errors.New("missing arguments"))
	}

	var (
		method       string
		targetURL    string
		headers      []entity.KeyValue
		dataParts    []string
		formParts    []string
		dataEncoded  []string
		useGetQuery  bool
	)

	addHeader := func(line string) {
		line = strings.TrimSpace(line)
		idx := strings.Index(line, ":")
		if idx <= 0 {
			return
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.TrimSpace(line[idx+1:])
		if k != "" {
			headers = append(headers, entity.KeyValue{Key: k, Value: v})
		}
	}

	for i := 0; i < len(args); i++ {
		a := args[i]

		switch {
		case a == "-X" || a == "--request":
			if i+1 < len(args) {
				i++
				method = strings.ToUpper(strings.TrimSpace(args[i]))
			}
		case a == "-H" || a == "--header":
			if i+1 < len(args) {
				i++
				addHeader(args[i])
			}
		case a == "-d" || a == "--data" || a == "--data-ascii" || a == "--data-raw" || a == "--data-binary":
			if i+1 < len(args) {
				i++
				s, rerr := readCurlDataArg(args[i])
				if rerr != nil {
					return nil, apperror.NewWithErrorDetail(constant.ErrInvalidCurl, rerr)
				}
				dataParts = append(dataParts, s)
			}
		case strings.HasPrefix(a, "--data="):
			s, rerr := readCurlDataArg(strings.TrimPrefix(a, "--data="))
			if rerr != nil {
				return nil, apperror.NewWithErrorDetail(constant.ErrInvalidCurl, rerr)
			}
			dataParts = append(dataParts, s)
		case strings.HasPrefix(a, "--data-raw="):
			s, rerr := readCurlDataArg(strings.TrimPrefix(a, "--data-raw="))
			if rerr != nil {
				return nil, apperror.NewWithErrorDetail(constant.ErrInvalidCurl, rerr)
			}
			dataParts = append(dataParts, s)
		case a == "--data-urlencode":
			if i+1 < len(args) {
				i++
				dataEncoded = append(dataEncoded, args[i])
			}
		case strings.HasPrefix(a, "--data-urlencode="):
			dataEncoded = append(dataEncoded, strings.TrimPrefix(a, "--data-urlencode="))
		case a == "-F" || a == "--form":
			if i+1 < len(args) {
				i++
				formParts = append(formParts, args[i])
			}
		case strings.HasPrefix(a, "-F") && len(a) > 2:
			formParts = append(formParts, strings.TrimPrefix(a, "-F"))
		case a == "--url":
			if i+1 < len(args) {
				i++
				targetURL = args[i]
			}
		case strings.HasPrefix(a, "--url="):
			targetURL = strings.TrimPrefix(a, "--url=")
		case a == "-b" || a == "--cookie":
			if i+1 < len(args) {
				i++
				addHeader("Cookie: " + args[i])
			}
		case a == "-A" || a == "--user-agent":
			if i+1 < len(args) {
				i++
				addHeader("User-Agent: " + args[i])
			}
		case a == "-u" || a == "--user":
			if i+1 < len(args) {
				i++
				up := args[i]
				b64 := base64.StdEncoding.EncodeToString([]byte(up))
				addHeader("Authorization: Basic " + b64)
			}
		case a == "-G" || a == "--get":
			useGetQuery = true
		case a == "-I" || a == "--head":
			method = "HEAD"
		case a == "-o" || a == "--output" || a == "-D" || a == "--dump-header" || a == "--trace-ascii" || a == "--trace":
			if i+1 < len(args) {
				i++
			}
		case strings.HasPrefix(a, "-"):
			continue
		default:
			if strings.HasPrefix(a, "http://") || strings.HasPrefix(a, "https://") {
				if targetURL == "" {
					targetURL = a
				}
			}
		}
	}

	for _, enc := range dataEncoded {
		dataParts = append(dataParts, enc)
	}

	if targetURL == "" {
		return nil, apperror.NewWithErrorDetail(constant.ErrInvalidCurl, errors.New("no URL found"))
	}

	if useGetQuery {
		method = "GET"
		if len(dataParts) > 0 {
			joined := strings.Join(dataParts, "&")
			u, err := url.Parse(targetURL)
			if err != nil {
				return nil, apperror.NewWithErrorDetail(constant.ErrInvalidCurl, err)
			}
			q := u.Query()
			extra, qerr := url.ParseQuery(joined)
			if qerr != nil {
				return nil, apperror.NewWithErrorDetail(constant.ErrInvalidCurl, qerr)
			}
			for k, vals := range extra {
				for _, v := range vals {
					q.Add(k, v)
				}
			}
			u.RawQuery = q.Encode()
			targetURL = u.String()
			dataParts = nil
		}
	}

	out := &entity.HTTPExecuteInput{
		Method:  method,
		URL:     targetURL,
		Headers: headers,
	}

	if len(formParts) > 0 {
		if out.Method == "" {
			out.Method = "POST"
		}
		parts, ferr := parseMultipartFormFlags(formParts)
		if ferr != nil {
			return nil, apperror.NewWithErrorDetail(constant.ErrInvalidCurl, ferr)
		}
		out.BodyMode = string(entity.BodyModeMultipartFormData)
		out.MultipartParts = parts
		return out, nil
	}

	if len(dataParts) == 0 {
		if out.Method == "" {
			out.Method = "GET"
		}
		out.BodyMode = string(entity.BodyModeNone)
		return out, nil
	}

	joined := strings.Join(dataParts, "&")
	ct := headerValueCI(headers, "Content-Type")
	ctLower := strings.ToLower(ct)

	if out.Method == "" {
		out.Method = "POST"
	}

	if strings.Contains(ctLower, "application/xml") || strings.Contains(ctLower, "text/xml") {
		out.BodyMode = string(entity.BodyModeXML)
		out.Body = joined
		return out, nil
	}

	if strings.HasPrefix(strings.TrimSpace(joined), "<?xml") {
		out.BodyMode = string(entity.BodyModeXML)
		out.Body = joined
		return out, nil
	}

	if strings.Contains(ctLower, "application/json") || looksLikeJSON(joined) {
		out.BodyMode = string(entity.BodyModeRaw)
		out.Body = joined
		return out, nil
	}

	if looksLikeFormEncoded(joined) {
		out.BodyMode = string(entity.BodyModeFormURLEncoded)
		fields, ferr := parseFormFieldsFromQueryString(joined)
		if ferr != nil || len(fields) == 0 {
			out.BodyMode = string(entity.BodyModeRaw)
			out.Body = joined
			return out, nil
		}
		out.FormFields = fields
		return out, nil
	}

	out.BodyMode = string(entity.BodyModeRaw)
	out.Body = joined
	return out, nil
}

func headerValueCI(headers []entity.KeyValue, name string) string {
	nl := strings.ToLower(name)
	for _, h := range headers {
		if strings.ToLower(strings.TrimSpace(h.Key)) == nl {
			return h.Value
		}
	}
	return ""
}

func looksLikeJSON(s string) bool {
	s = strings.TrimSpace(s)
	return len(s) > 0 && (s[0] == '{' || s[0] == '[')
}

func looksLikeFormEncoded(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || looksLikeJSON(s) {
		return false
	}
	return strings.Contains(s, "=")
}

func parseFormFieldsFromQueryString(s string) ([]entity.KeyValue, error) {
	v, err := url.ParseQuery(strings.TrimSpace(s))
	if err != nil {
		return nil, err
	}
	var out []entity.KeyValue
	for k, vals := range v {
		for _, val := range vals {
			out = append(out, entity.KeyValue{Key: k, Value: val})
		}
	}
	return out, nil
}

func readCurlDataArg(s string) (string, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "@") {
		path := strings.TrimPrefix(s, "@")
		path = strings.TrimSpace(path)
		if path == "" || path == "-" {
			return "", errors.New("stdin not supported; use @file path")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return s, nil
}

func parseMultipartFormFlags(formParts []string) ([]entity.MultipartPart, error) {
	var out []entity.MultipartPart
	for _, fp := range formParts {
		idx := strings.Index(fp, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(fp[:idx])
		val := fp[idx+1:]
		if key == "" {
			continue
		}
		if strings.HasPrefix(val, "@") {
			path := strings.TrimPrefix(val, "@")
			path = strings.TrimSpace(path)
			if path == "" || path == "-" {
				return nil, errors.New("multipart file path required after @")
			}
			out = append(out, entity.MultipartPart{
				Key:      key,
				Kind:     "file",
				FilePath: path,
			})
		} else {
			out = append(out, entity.MultipartPart{
				Key:   key,
				Kind:  "text",
				Value: val,
			})
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no form fields parsed from -F")
	}
	return out, nil
}
