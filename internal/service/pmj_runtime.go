package service

import (
	"PostmanJanai/internal/entity"
	"PostmanJanai/internal/pkg/logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
)

type pmjExecutor interface {
	Execute(ctx context.Context, in *entity.HTTPExecuteInput) (*entity.HTTPExecuteResult, error)
}

type pmjEnv interface {
	ActiveVariableMap(ctx context.Context) (map[string]string, error)
	UpsertActiveVariable(ctx context.Context, key, value string) (bool, error)
	DeleteActiveVariable(ctx context.Context, key string) (bool, error)
}

// PMJArtifacts captures console + test assertions recorded from sandboxed scripts.
type PMJArtifacts struct {
	Console []entity.ScriptConsoleLine `json:"console,omitempty"`
	Tests   []entity.ScriptTestResult  `json:"tests,omitempty"`
}

// RunPMJScript evaluates `scriptText` in a fresh goja runtime with globals `pmj` and `pm` (alias).
func RunPMJScript(
	ctx context.Context,
	isPrePhase bool,
	scriptText string,
	timeout time.Duration,
	req *entity.HTTPExecuteInput,
	res *entity.HTTPExecuteResult,
	session map[string]string,
	env pmjEnv,
	executor pmjExecutor,
) (*PMJArtifacts, error) {
	if strings.TrimSpace(scriptText) == "" {
		return &PMJArtifacts{}, nil
	}
	if executor == nil {
		return nil, errors.New("http executor unavailable for scripting")
	}
	vm := goja.New()
	timer := time.AfterFunc(timeout, func() {
		vm.Interrupt("script timeout")
	})
	defer timer.Stop()

	out := &PMJArtifacts{}
	bindConsole(vm, out)
	if session == nil {
		session = map[string]string{}
	}
	st := &pmjSandboxState{
		ctx:           ctx,
		vm:            vm,
		env:           env,
		session:       session,
		httpExecutor:  executor,
		out:           out,
		activeRequest: req,
		activeResp:    res,
		isPrePhase:    isPrePhase,
	}
	if err := st.installGlobals(); err != nil {
		return nil, err
	}

	done := make(chan struct{})
	var runErr error
	go func() {
		defer close(done)
		defer func() {
			if pv := recover(); pv != nil {
				runErr = fmt.Errorf("%v", pv)
			}
		}()
		_, runErr = vm.RunString(strings.TrimPrefix(scriptText, "\ufeff"))
	}()

	select {
	case <-done:
	case <-ctx.Done():
		vm.Interrupt("cancelled")
		<-done
		if runErr == nil || isInterrupt(runErr) {
			runErr = ctx.Err()
		}
	}
	if runErr != nil {
		if isInterrupt(runErr) {
			return nil, fmt.Errorf("%w (%s)", runErr, scriptPhaseLabel(isPrePhase))
		}
		return nil, fmt.Errorf("%s: %w", scriptPhaseLabel(isPrePhase), runErr)
	}
	return out, nil
}

func scriptPhaseLabel(pre bool) string {
	if pre {
		return "pre-request script"
	}
	return "post-response script"
}

func isInterrupt(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "interrupt")
}

func bindConsole(rt *goja.Runtime, artifacts *PMJArtifacts) {
	if artifacts == nil {
		return
	}
	var mu sync.Mutex
	push := func(level, msg string) {
		level = strings.ToLower(strings.TrimSpace(level))
		if level == "" {
			level = "log"
		}
		mu.Lock()
		artifacts.Console = append(artifacts.Console, entity.ScriptConsoleLine{Level: level, Message: msg})
		mu.Unlock()
	}
	c := rt.NewObject()
	_ = c.Set("log", func(call goja.FunctionCall) { push("log", concatArgs(rt, call)) })
	_ = c.Set("info", func(call goja.FunctionCall) { push("info", concatArgs(rt, call)) })
	_ = c.Set("warn", func(call goja.FunctionCall) { push("warn", concatArgs(rt, call)) })
	_ = c.Set("error", func(call goja.FunctionCall) { push("error", concatArgs(rt, call)) })
	_ = c.Set("debug", func(call goja.FunctionCall) { push("debug", concatArgs(rt, call)) })
	_ = rt.Set("console", c)
}

func concatArgs(rt *goja.Runtime, call goja.FunctionCall) string {
	var b strings.Builder
	for i := range call.Arguments {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(gojaFmt(rt, call.Argument(i)))
	}
	return b.String()
}

func gojaFmt(rt *goja.Runtime, v goja.Value) string {
	if v == nil || goja.IsUndefined(v) {
		return ""
	}
	exported := v.Export()
	switch t := exported.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case nil:
		return v.String()
	default:
		if b, err := json.Marshal(t); err == nil {
			return string(b)
		}
	}
	return v.String()
}

type pmjSandboxState struct {
	ctx           context.Context
	vm            *goja.Runtime
	env           pmjEnv
	session       map[string]string
	httpExecutor  pmjExecutor
	out           *PMJArtifacts
	activeRequest *entity.HTTPExecuteInput
	activeResp    *entity.HTTPExecuteResult
	isPrePhase    bool
}

func (s *pmjSandboxState) setFn(obj *goja.Object, name string, fn func(goja.FunctionCall) goja.Value) error {
	return obj.Set(name, fn)
}

func (s *pmjSandboxState) installGlobals() error {
	vm := s.vm
	if vm == nil {
		return errors.New("nil VM")
	}
	pmj := vm.NewObject()

	if err := s.bindVariablesSubtree(pmj); err != nil {
		return err
	}
	if err := s.bindEnvironmentSubtree(pmj); err != nil {
		return err
	}
	if err := vm.Set("pmj", pmj); err != nil {
		return err
	}
	if err := s.attachTestAPI(); err != nil {
		return err
	}

	if err := s.mountRequestSubtree(pmj); err != nil {
		return err
	}
	if !s.isPrePhase {
		if err := s.mountResponseSubtree(pmj); err != nil {
			return err
		}
	}
	if err := s.mountSendRequest(pmj); err != nil {
		return err
	}
	if _, err := vm.RunString("globalThis.pm = globalThis.pmj"); err != nil {
		return err
	}
	return nil
}

func (s *pmjSandboxState) bindVariablesSubtree(pmj *goja.Object) error {
	vm := s.vm
	v := vm.NewObject()
	_ = s.setFn(v, "get", func(c goja.FunctionCall) goja.Value {
		k := strings.TrimSpace(c.Argument(0).String())
		if k == "" {
			return vm.ToValue("")
		}
		if x, ok := s.session[k]; ok {
			return vm.ToValue(x)
		}
		if s.env != nil && s.ctx != nil {
			if m, err := s.env.ActiveVariableMap(s.ctx); err == nil {
				return vm.ToValue(m[k])
			}
		}
		return vm.ToValue("")
	})
	_ = s.setFn(v, "set", func(c goja.FunctionCall) goja.Value {
		k := strings.TrimSpace(c.Argument(0).String())
		if k != "" {
			s.session[k] = gojaFmt(vm, c.Argument(1))
		}
		return goja.Undefined()
	})
	_ = s.setFn(v, "unset", func(c goja.FunctionCall) goja.Value {
		k := strings.TrimSpace(c.Argument(0).String())
		if k != "" {
			delete(s.session, k)
		}
		return goja.Undefined()
	})
	return pmj.Set("variables", v)
}

func (s *pmjSandboxState) bindEnvironmentSubtree(pmj *goja.Object) error {
	vm := s.vm
	e := vm.NewObject()
	_ = s.setFn(e, "get", func(c goja.FunctionCall) goja.Value {
		k := strings.TrimSpace(c.Argument(0).String())
		if k == "" || s.env == nil || s.ctx == nil {
			return vm.ToValue("")
		}
		m, err := s.env.ActiveVariableMap(s.ctx)
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue(m[k])
	})
	_ = s.setFn(e, "set", func(c goja.FunctionCall) goja.Value {
		k := strings.TrimSpace(c.Argument(0).String())
		if k == "" || s.env == nil || s.ctx == nil {
			return goja.Undefined()
		}
		val := gojaFmt(vm, c.Argument(1))
		if _, err := s.env.UpsertActiveVariable(s.ctx, k, val); err != nil {
			logger.L().WarnContext(s.ctx, "pmj.environment.set", "error", err)
		}
		s.session[k] = val
		return goja.Undefined()
	})
	_ = s.setFn(e, "unset", func(c goja.FunctionCall) goja.Value {
		k := strings.TrimSpace(c.Argument(0).String())
		if k == "" || s.env == nil || s.ctx == nil {
			return goja.Undefined()
		}
		if _, err := s.env.DeleteActiveVariable(s.ctx, k); err != nil {
			logger.L().WarnContext(s.ctx, "pmj.environment.unset", "error", err)
		}
		delete(s.session, k)
		return goja.Undefined()
	})
	return pmj.Set("environment", e)
}

func (s *pmjSandboxState) attachTestAPI() error {
	vm := s.vm
	if err := vm.Set("__pmjRecordTest", func(name string, ok bool, detail string) {
		name = strings.TrimSpace(name)
		if name == "" {
			name = "unnamed test"
		}
		s.out.Tests = append(s.out.Tests, entity.ScriptTestResult{Name: name, Passed: ok, Detail: strings.TrimSpace(detail)})
	}); err != nil {
		return err
	}
	if err := vm.Set("__pmjDeepEq", func(a, b goja.Value) bool {
		return reflect.DeepEqual(a.Export(), b.Export())
	}); err != nil {
		return err
	}

	bootstrap := `
(function(){
  var pmRoot = globalThis.pmj;
  pmRoot.test = function(name, fn){
    try {
      fn();
      __pmjRecordTest(String(name||''), true, '');
    } catch(e) {
      __pmjRecordTest(String(name||''), false, String(e && e.message ? e.message : e));
      throw e;
    }
  };
  pmRoot.expect = function(actual){
    return {
      to: {
        equal: function(exp){
          if (actual !== exp) { throw new Error('expected '+String(exp)+' got '+String(actual)); }
        },
        eql: function(exp){ if(!__pmjDeepEq(actual,exp)) throw new Error('deep equal failed'); },
        include: function(x){
          if(String(actual).indexOf(String(x))<0){ throw new Error('include failed'); }
        },
        exist: function(){
          var bad = actual===null||actual===undefined||actual==='';
          if(bad) throw new Error('expected existence');
        },
        get be(){
          var a = actual;
          return {
            ok: function(){ if (!a){ throw new Error('not truthy'); } },
            true: function(){ if (a !== true){ throw new Error('not true'); }},
            false: function(){ if (a !== false){ throw new Error('not false'); }},
            get a(){ return function(t){
              var want = String(t||'').replace(/['"]/g,'');
              var got = typeof a;
              if (want === 'string' && got !== 'string') { throw new Error('type mismatch'); }
              if (want === 'number' && got !== 'number') { throw new Error('type mismatch'); }
            };}
          };
        },
        get have(){ return {
          status: function(c){
            var want = typeof c === 'string' ? parseInt(String(c),10) : c;
            if (actual !== want) throw new Error('bad status');
          }
        };}
      }
    };
  };
})();
`
	if _, err := vm.RunString(bootstrap); err != nil {
		return fmt.Errorf("pmj bootstrap: %w", err)
	}
	return nil
}

func (s *pmjSandboxState) mountRequestSubtree(pmj *goja.Object) error {
	vm := s.vm

	rqGetMethod := func() string {
		if s.activeRequest == nil {
			return http.MethodGet
		}
		m := strings.TrimSpace(strings.ToUpper(s.activeRequest.Method))
		if m == "" {
			return http.MethodGet
		}
		return m
	}
	rqSetMethod := func(v string) {
		if s.activeRequest == nil {
			return
		}
		if strings.TrimSpace(v) == "" {
			s.activeRequest.Method = http.MethodGet
			return
		}
		s.activeRequest.Method = strings.TrimSpace(strings.ToUpper(v))
	}
	rqGetURL := func() string {
		if s.activeRequest == nil {
			return ""
		}
		return strings.TrimSpace(s.activeRequest.URL)
	}
	rqSetURL := func(v string) {
		if s.activeRequest != nil {
			s.activeRequest.URL = v
		}
	}
	rqHdrGet := func(key string) string {
		key = strings.TrimSpace(key)
		if key == "" || s.activeRequest == nil {
			return ""
		}
		lk := strings.ToLower(key)
		for _, h := range s.activeRequest.Headers {
			if strings.ToLower(strings.TrimSpace(h.Key)) == lk {
				return h.Value
			}
		}
		return ""
	}
	rqHdrAdd := func(key, val string) {
		if s.activeRequest == nil {
			return
		}
		k := strings.TrimSpace(key)
		if k == "" {
			return
		}
		s.activeRequest.Headers = append(s.activeRequest.Headers, entity.KeyValue{Key: k, Value: val})
	}
	rqBodyGet := func() string {
		if s.activeRequest == nil {
			return ""
		}
		return s.activeRequest.Body
	}
	rqBodySet := func(v string) {
		if s.activeRequest != nil {
			s.activeRequest.Body = v
		}
	}

	hooks := map[string]interface{}{
		"__rqGetMethod": rqGetMethod,
		"__rqSetMethod": rqSetMethod,
		"__rqGetURL":    rqGetURL,
		"__rqSetURL":    rqSetURL,
		"__rqHdrGet":    rqHdrGet,
		"__rqHdrAdd":    rqHdrAdd,
		"__rqBodyGet":   rqBodyGet,
		"__rqBodySet":   rqBodySet,
	}
	for k, v := range hooks {
		if err := vm.Set(k, v); err != nil {
			return err
		}
	}

	install := `
(() => {
  const r = {};
  Object.defineProperty(r, 'method', {
    get() { return __rqGetMethod(); },
    set(v) { __rqSetMethod(String(v || '').trim()); },
    enumerable: true,
  });
  Object.defineProperty(r, 'url', {
    get() { return __rqGetURL(); },
    set(v) { __rqSetURL(String(v || '')); },
    enumerable: true,
  });
  Object.defineProperty(r, 'headers', {
    value: {
      get: function(k) { return __rqHdrGet(String(k||'')); },
      add: function(k, v) { __rqHdrAdd(k, v); },
    },
    enumerable: true,
  });
  const bd = {};
  Object.defineProperty(bd, 'raw', {
    get() { return __rqBodyGet(); },
    set(v) { __rqBodySet(v == null ? '' : String(v)); },
    enumerable: true,
  });
  r.body = bd;
  __pmMountReq(r);
})();
`
	if err := vm.Set("__pmMountReq", func(c goja.FunctionCall) goja.Value {
		_ = pmj.Set("request", c.Argument(0))
		return goja.Undefined()
	}); err != nil {
		return err
	}
	_, err := vm.RunString(install)
	return err
}

func (s *pmjSandboxState) mountResponseSubtree(pmj *goja.Object) error {
	vm := s.vm

	if err := vm.Set("__rsStatus", func() int {
		if s.activeResp == nil {
			return 0
		}
		return s.activeResp.StatusCode
	}); err != nil {
		return err
	}
	if err := vm.Set("__rsHeadersGet", func(k string) string {
		key := strings.TrimSpace(k)
		if key == "" || s.activeResp == nil {
			return ""
		}
		lk := strings.ToLower(key)
		for _, h := range s.activeResp.ResponseHeaders {
			if strings.ToLower(strings.TrimSpace(h.Key)) == lk {
				return h.Value
			}
		}
		return ""
	}); err != nil {
		return err
	}
	if err := vm.Set("__rsText", func() string {
		if s.activeResp == nil {
			return ""
		}
		return s.activeResp.ResponseBody
	}); err != nil {
		return err
	}
	if err := vm.Set("__rsDur", func() int64 {
		if s.activeResp == nil {
			return 0
		}
		return s.activeResp.DurationMs
	}); err != nil {
		return err
	}
	if err := vm.Set("__rsURL", func() string {
		if s.activeResp == nil {
			return ""
		}
		return strings.TrimSpace(s.activeResp.FinalURL)
	}); err != nil {
		return err
	}
	if err := vm.Set("__rsJSONGo", func() goja.Value {
		if s.activeResp == nil {
			panic(vm.NewGoError(errors.New("no response")))
		}
		raw := strings.TrimSpace(s.activeResp.ResponseBody)
		if raw == "" {
			panic(vm.NewGoError(errors.New("empty response body")))
		}
		var decoded interface{}
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			panic(vm.NewGoError(fmt.Errorf("json parse: %w", err)))
		}
		return vm.ToValue(decoded)
	}); err != nil {
		return err
	}

	fixed := `
(() => {
  const respHeaders = { get: function(k) { return __rsHeadersGet(k); } };
  const ro = {};
  Object.defineProperty(ro,'code',{get:function(){return __rsStatus()}, enumerable:true});
  Object.defineProperty(ro,'responseTime',{get:function(){return __rsDur()}, enumerable:true});
  Object.defineProperty(ro,'url',{get:function(){return __rsURL()}, enumerable:true});
  Object.defineProperty(ro,'headers',{value:respHeaders, enumerable:true});
  ro.text=function(){return __rsText()};
  ro.json=function(){return __rsJSONGo(); };
  __pmMountResp(ro);
})();
`
	if err := vm.Set("__pmMountResp", func(c goja.FunctionCall) goja.Value {
		_ = pmj.Set("response", c.Argument(0))
		return goja.Undefined()
	}); err != nil {
		return err
	}
	_, err := vm.RunString(fixed)
	return err
}

func (s *pmjSandboxState) mountSendRequest(pmj *goja.Object) error {
	vm := s.vm
	return s.setFn(pmj, "sendRequest", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("sendRequest requires options"))
		}
		opts := call.Argument(0)
		cb := goja.Undefined()
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Argument(1)) {
			cb = call.Argument(1)
		}
		sub := s.buildNestedExecuteInput(opts.Export())
		if sub == nil {
			panic(vm.ToValue("invalid sendRequest options"))
		}
		parent := s.activeRequest
		var root *string
		if parent != nil {
			root = parent.RootFolderID
		}
		sub.RootFolderID = root
		sub.RequestID = nil
		MergeAuthIntoHeadersAndQuery(sub)
		res, execErr := s.httpExecutor.Execute(s.ctx, sub)
		respObj := s.responseToJS(vm, res, execErr)

		if !goja.IsUndefined(cb) {
			fn, ok := goja.AssertFunction(cb)
			if ok {
				if execErr != nil {
					if _, e := fn(goja.Null(), vm.ToValue(execErr.Error()), respObj); e != nil {
						panic(e)
					}
				} else {
					if _, e := fn(goja.Null(), goja.Null(), respObj); e != nil {
						panic(e)
					}
				}
			}
		}
		return goja.Undefined()
	})
}

func (s *pmjSandboxState) buildNestedExecuteInput(exp interface{}) *entity.HTTPExecuteInput {
	row, ok := exp.(map[string]interface{})
	if !ok || row == nil {
		return nil
	}
	u := ""
	if x, ok := row["url"].(string); ok {
		u = strings.TrimSpace(x)
	}
	if u == "" {
		return nil
	}
	method := http.MethodGet
	if m, ok := row["method"].(string); ok && strings.TrimSpace(m) != "" {
		method = strings.ToUpper(strings.TrimSpace(m))
	}
	sub := &entity.HTTPExecuteInput{Method: method, URL: u, BodyMode: string(entity.BodyModeNone)}
	if b, ok := row["body"].(string); ok {
		sub.BodyMode = string(entity.BodyModeRaw)
		sub.Body = b
	}
	if hdrs, ok := row["headers"]; ok {
		switch h := hdrs.(type) {
		case map[string]interface{}:
			for k, v := range h {
				k = strings.TrimSpace(k)
				if k == "" {
					continue
				}
				sub.Headers = append(sub.Headers, entity.KeyValue{Key: k, Value: fmt.Sprint(v)})
			}
		case []interface{}:
			for _, it := range h {
				mm, ok := it.(map[string]interface{})
				if !ok {
					continue
				}
				key := strings.TrimSpace(fmt.Sprint(mm["key"]))
				if key == "" {
					continue
				}
				sub.Headers = append(sub.Headers, entity.KeyValue{Key: key, Value: fmt.Sprint(mm["value"])})
			}
		}
	}
	if q, ok := row["query"].([]interface{}); ok {
		for _, it := range q {
			mm, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			k := strings.TrimSpace(fmt.Sprint(mm["key"]))
			if k == "" {
				continue
			}
			sub.QueryParams = append(sub.QueryParams, entity.KeyValue{Key: k, Value: fmt.Sprint(mm["value"])})
		}
	}
	return sub
}

func (s *pmjSandboxState) responseToJS(vm *goja.Runtime, res *entity.HTTPExecuteResult, err error) goja.Value {
	if err != nil || res == nil {
		o := vm.NewObject()
		_ = o.Set("code", func(goja.FunctionCall) goja.Value { return vm.ToValue(0) })
		_ = o.Set("headers", vm.NewObject())
		_ = o.Set("text", func(goja.FunctionCall) goja.Value {
			if err != nil {
				return vm.ToValue(err.Error())
			}
			return vm.ToValue("")
		})
		_ = o.Set("json", func(goja.FunctionCall) goja.Value { panic(vm.ToValue("invalid json")) })
		_ = o.Set("responseTime", func(goja.FunctionCall) goja.Value { return vm.ToValue(0) })
		_ = o.Set("url", func(goja.FunctionCall) goja.Value { return vm.ToValue("") })
		return o
	}
	headerObj := vm.NewObject()
	hdrGet := func(c goja.FunctionCall) goja.Value {
		k := strings.TrimSpace(c.Argument(0).String())
		if k == "" {
			return vm.ToValue("")
		}
		lk := strings.ToLower(k)
		for _, h := range res.ResponseHeaders {
			if strings.EqualFold(strings.TrimSpace(h.Key), lk) {
				return vm.ToValue(h.Value)
			}
		}
		return vm.ToValue("")
	}
	_ = headerObj.Set("get", hdrGet)

	bodyTxt := ""
	if res != nil {
		bodyTxt = res.ResponseBody
	}
	jsonFn := func(goja.FunctionCall) goja.Value {
		raw := strings.TrimSpace(bodyTxt)
		if raw == "" {
			panic(vm.ToValue(errors.New("empty")))
		}
		var decoded interface{}
		if e := json.Unmarshal([]byte(raw), &decoded); e != nil {
			panic(vm.ToValue(e.Error()))
		}
		return vm.ToValue(decoded)
	}

	o := vm.NewObject()
	_ = o.Set("code", func(goja.FunctionCall) goja.Value {
		return vm.ToValue(res.StatusCode)
	})
	if res.ErrorMessage != "" {
		if res.StatusCode <= 0 {
			_ = o.Set("code", func(goja.FunctionCall) goja.Value { return vm.ToValue(0) })
		}
	}
	_ = o.Set("headers", headerObj)
	_ = o.Set("text", func(goja.FunctionCall) goja.Value {
		t := ""
		if res != nil && res.ResponseBody != "" {
			t = res.ResponseBody
		}
		if res != nil && t == "" && res.ErrorMessage != "" {
			t = res.ErrorMessage
		}
		return vm.ToValue(t)
	})
	_ = o.Set("json", jsonFn)
	_ = o.Set("responseTime", func(goja.FunctionCall) goja.Value {
		if res != nil && res.DurationMs > 0 {
			return vm.ToValue(int(res.DurationMs))
		}
		return vm.ToValue(0)
	})
	_ = o.Set("url", func(goja.FunctionCall) goja.Value {
		u := ""
		if res != nil {
			u = strings.TrimSpace(res.FinalURL)
		}
		return vm.ToValue(u)
	})
	return o
}
