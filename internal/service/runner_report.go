package service

import (
	"PostmanJanai/internal/entity"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// MarshalRunnerRunDetailJSON returns the indented JSON form used by the
// "Export run as JSON" menu in the Runner modal.
func MarshalRunnerRunDetailJSON(d *entity.RunnerRunDetail) ([]byte, error) {
	if d == nil {
		return []byte("{}"), nil
	}
	return json.MarshalIndent(d, "", "  ")
}

// MarshalRunnerRunDetailMarkdown produces a human-friendly Markdown report.
//
// Layout choices:
//   - Top-level header includes folder + env + status counts (handy for PR descriptions).
//   - Per-request section includes both assertions + captures so reviewers can
//     audit chains without re-running.
//   - Long bodies / values are truncated at 600 chars to keep the markdown
//     reasonable in chat clients.
func MarshalRunnerRunDetailMarkdown(d *entity.RunnerRunDetail) []byte {
	if d == nil {
		return []byte("# Runner report\n\n_No data._")
	}
	var b strings.Builder
	b.WriteString("# Runner report\n\n")
	b.WriteString(fmt.Sprintf("- **Folder:** %s\n", fallback(d.FolderName, "(unnamed)")))
	if d.EnvironmentName != "" {
		b.WriteString(fmt.Sprintf("- **Environment:** %s\n", d.EnvironmentName))
	}
	b.WriteString(fmt.Sprintf("- **Status:** `%s`\n", fallback(d.Status, "—")))
	b.WriteString(fmt.Sprintf("- **Started:** %s\n", d.StartedAt.UTC().Format(time.RFC3339)))
	if d.FinishedAt != nil {
		b.WriteString(fmt.Sprintf("- **Finished:** %s\n", d.FinishedAt.UTC().Format(time.RFC3339)))
	}
	b.WriteString(fmt.Sprintf(
		"- **Totals:** total=%d · passed=%d · failed=%d · errored=%d · duration=%dms\n",
		d.TotalCount, d.PassedCount, d.FailedCount, d.ErrorCount, d.DurationMs,
	))
	if strings.TrimSpace(d.Notes) != "" {
		b.WriteString(fmt.Sprintf("- **Notes:** %s\n", strings.TrimSpace(d.Notes)))
	}
	b.WriteString("\n## Requests\n\n")
	if len(d.Requests) == 0 {
		b.WriteString("_No requests recorded._\n")
		return []byte(b.String())
	}
	b.WriteString("| # | Status | Method | Request | Code | Duration | Size |\n")
	b.WriteString("|---|--------|--------|---------|------|----------|------|\n")
	for i, r := range d.Requests {
		b.WriteString(fmt.Sprintf(
			"| %d | %s | %s | %s | %d | %d ms | %d B |\n",
			i+1,
			r.Status,
			fallback(r.Method, "—"),
			escapeMD(fmt.Sprintf("%s — %s", r.RequestName, r.URL)),
			r.StatusCode,
			r.DurationMs,
			r.ResponseSizeBytes,
		))
	}

	b.WriteString("\n## Details\n")
	for i, r := range d.Requests {
		b.WriteString(fmt.Sprintf("\n### %d. %s `%s` `%s`\n",
			i+1, fallback(r.RequestName, "(unnamed)"), strings.ToUpper(r.Method), r.URL))
		b.WriteString(fmt.Sprintf("- Status: `%s` (HTTP %d) · %d ms · %d B\n",
			r.Status, r.StatusCode, r.DurationMs, r.ResponseSizeBytes))
		if r.ErrorMessage != "" {
			b.WriteString(fmt.Sprintf("- Error: `%s`\n", truncate(r.ErrorMessage, 600)))
		}
		if len(r.Assertions) > 0 {
			b.WriteString("\n**Assertions**\n\n")
			for _, a := range r.Assertions {
				mark := "PASS"
				if !a.Passed {
					mark = "FAIL"
				}
				b.WriteString(fmt.Sprintf("- [%s] `%s` (`%s` %s `%s`) — actual=`%s`",
					mark,
					escapeMD(fallback(a.Name, "(unnamed)")),
					a.Source, a.Operator,
					escapeMD(truncate(a.Expected, 200)),
					escapeMD(truncate(a.Actual, 200)),
				))
				if a.ErrorMessage != "" {
					b.WriteString(fmt.Sprintf(" — error: %s", escapeMD(truncate(a.ErrorMessage, 200))))
				}
				b.WriteString("\n")
			}
		}
		if len(r.Captures) > 0 {
			b.WriteString("\n**Captures**\n\n")
			for _, c := range r.Captures {
				ok := "OK"
				if c.ErrorMessage != "" {
					ok = "ERR"
				} else if !c.Captured {
					ok = "—"
				}
				b.WriteString(fmt.Sprintf("- [%s] `%s` → `%s.%s` (`%s`) value=`%s`",
					ok,
					escapeMD(fallback(c.Name, "(unnamed)")),
					c.TargetScope, c.TargetVariable,
					c.Source,
					escapeMD(truncate(c.Value, 200)),
				))
				if c.ErrorMessage != "" {
					b.WriteString(fmt.Sprintf(" — error: %s", escapeMD(truncate(c.ErrorMessage, 200))))
				}
				b.WriteString("\n")
			}
		}
	}
	return []byte(b.String())
}

func fallback(s, def string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return def
	}
	return t
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func escapeMD(s string) string {
	r := strings.NewReplacer(
		"|", "\\|",
		"\r\n", " ",
		"\n", " ",
		"\r", " ",
	)
	return r.Replace(s)
}
