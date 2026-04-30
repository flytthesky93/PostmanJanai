package service

import (
	"PostmanJanai/internal/entity"
	"strings"
)

// PMJAPIToken is the canonical script global for PostmanJanai scripting.
const PMJAPIToken = "pmj."

// ReplacePostmanPMAliasWithPMJ rewrites legacy Postman-style `pm.` references to `pmj.`
// so imports stay consistent with the in-app scripting API prefix.
func ReplacePostmanPMAliasWithPMJ(script string) string {
	if strings.TrimSpace(script) == "" {
		return script
	}
	return strings.ReplaceAll(script, "pm.", PMJAPIToken)
}

// ReplacePMJWithPostmanPMAlias rewrites exported scripts for Postman Collection v2.1 interop.
func ReplacePMJWithPostmanPMAlias(script string) string {
	if strings.TrimSpace(script) == "" {
		return script
	}
	return strings.ReplaceAll(script, PMJAPIToken, "pm.")
}

// PostmanEventListenPrerequest / PostmanEventListenTest — Postman collection event listens.
const (
	PostmanEventListenPrerequest = "prerequest"
	PostmanEventListenTest       = "test"
)

func postmanEventBlock(listen string, script string) map[string]interface{} {
	exec := strings.Split(strings.ReplaceAll(script, "\r\n", "\n"), "\n")
	return map[string]interface{}{
		"listen": strings.TrimSpace(strings.ToLower(listen)),
		"script": map[string]interface{}{
			"type": "text/javascript",
			"exec": exec,
		},
	}
}

// PostmanExportRequestEvents emits Postman-compatible `event` blocks for Collection v2.1 export.
func PostmanExportRequestEvents(full *entity.SavedRequestFull) []interface{} {
	if full == nil {
		return nil
	}
	var out []interface{}
	if s := strings.TrimSpace(full.PreRequestScript); s != "" {
		out = append(out, postmanEventBlock(PostmanEventListenPrerequest, ReplacePMJWithPostmanPMAlias(s)))
	}
	if s := strings.TrimSpace(full.PostResponseScript); s != "" {
		out = append(out, postmanEventBlock(PostmanEventListenTest, ReplacePMJWithPostmanPMAlias(s)))
	}
	return out
}
