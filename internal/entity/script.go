package entity

// ScriptConsoleLine — one line bridged from `console.*` inside a goja sandbox.
type ScriptConsoleLine struct {
	Level   string `json:"level"` // info, log, warn, error, debug
	Message string `json:"message"`
}

// ScriptTestResult — outcome of `pmj.test("name", fn)` (subset tests).
type ScriptTestResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Detail  string `json:"detail,omitempty"`
}
