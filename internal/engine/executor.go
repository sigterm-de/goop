package engine

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"codeberg.org/sigterm-de/goop/internal/logging"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// errTimeout is the interrupt value used to distinguish a timeout from other
// interrupt causes.
var errTimeout = errors.New("script execution timed out")

// executor is the production implementation of the Executor interface.
// It is safe to call from multiple goroutines simultaneously; each Execute call
// creates an independent goja runtime.
type executor struct{}

// NewExecutor returns a ready-to-use Executor.
func NewExecutor() Executor {
	return &executor{}
}

// Execute runs a single Boop script against the provided input and returns a
// structured result. It never panics.
func (e *executor) Execute(ctx context.Context, input ExecutionInput) (result ExecutionResult) {
	// Recover from any internal panic.
	defer func() {
		if r := recover(); r != nil {
			result = ExecutionResult{
				Success:      false,
				ScriptName:   input.ScriptName,
				ErrorMessage: fmt.Sprintf("internal engine error: %v", r),
			}
		}
	}()

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	// ── Sandbox: poison prohibited globals ───────────────────────────────────
	poisoned := []string{
		"fetch", "XMLHttpRequest", "WebSocket",
		"process", "global", "Buffer",
		"setTimeout", "setInterval", "clearTimeout", "clearInterval",
	}
	for _, name := range poisoned {
		vm.Set(name, goja.Undefined())
	}

	// ── Module system: only @boop/ paths ─────────────────────────────────────
	registry := require.NewRegistry(require.WithLoader(blockingRequireLoader))
	registerModules(registry)
	registry.Enable(vm)

	// ── Additional globals ───────────────────────────────────────────────────
	registerBtoaAtob(vm)
	registerConsoleLog(vm, input.ScriptName)

	// ── State object ─────────────────────────────────────────────────────────
	state := NewScriptState(input)
	if err := bindState(vm, state); err != nil {
		return ExecutionResult{
			Success:      false,
			ScriptName:   input.ScriptName,
			ErrorMessage: fmt.Sprintf("internal engine error: bind state: %v", err),
		}
	}

	// ── Compile for syntax check (before starting timer) ─────────────────────
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	prog, err := goja.Compile(input.ScriptName, input.ScriptSource, false)
	if err != nil {
		return ExecutionResult{
			Success:      false,
			ScriptName:   input.ScriptName,
			ErrorMessage: err.Error(),
		}
	}

	// ── Timeout timer ─────────────────────────────────────────────────────────
	timedOut := false
	timer := time.AfterFunc(timeout, func() {
		timedOut = true
		vm.Interrupt(errTimeout)
	})
	defer timer.Stop()

	// ── Context cancellation ──────────────────────────────────────────────────
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			timedOut = true
			vm.Interrupt(errTimeout)
		case <-stop:
		}
	}()

	// ── Run: define functions + call main(state) ─────────────────────────────
	if _, runErr := vm.RunProgram(prog); runErr != nil {
		return e.runError(runErr, timedOut, timeout, input.ScriptName)
	}

	// Retrieve and call main(state)
	mainFn, ok := goja.AssertFunction(vm.Get("main"))
	if !ok {
		return ExecutionResult{
			Success:      false,
			ScriptName:   input.ScriptName,
			ErrorMessage: "script does not define a top-level function main(state)",
		}
	}

	stateVal := vm.Get("state")
	if _, callErr := mainFn(goja.Undefined(), stateVal); callErr != nil {
		return e.runError(callErr, timedOut, timeout, input.ScriptName)
	}

	return state.Result(input.ScriptName)
}

func (e *executor) runError(err error, timedOut bool, timeout time.Duration, scriptName string) ExecutionResult {
	if timedOut {
		return ExecutionResult{
			Success:      false,
			TimedOut:     true,
			ScriptName:   scriptName,
			ErrorMessage: fmt.Sprintf("Script execution timed out after %v", timeout),
		}
	}
	msg := err.Error()
	var jsException *goja.Exception
	if errors.As(err, &jsException) {
		msg = jsException.Error()
	}
	return ExecutionResult{
		Success:      false,
		ScriptName:   scriptName,
		ErrorMessage: msg,
	}
}

// bindState exposes the state object to the JS VM with property setters that
// track which fields are mutated.
func bindState(vm *goja.Runtime, state *ScriptState) error {
	stateObj := vm.NewObject()

	// fullText property with mutation tracking
	if err := stateObj.DefineAccessorProperty("fullText",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(state.FullText)
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			state.FullText = call.Arguments[0].String()
			state.fullTextMutated = true
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	); err != nil {
		return fmt.Errorf("fullText property: %w", err)
	}

	// text property with mutation tracking
	if err := stateObj.DefineAccessorProperty("text",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(state.Text)
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			state.Text = call.Arguments[0].String()
			state.textMutated = true
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	); err != nil {
		return fmt.Errorf("text property: %w", err)
	}

	// selection (read-only)
	selObj := vm.NewObject()
	selObj.Set("start", state.SelectionInfo.Start)
	selObj.Set("end", state.SelectionInfo.End)
	stateObj.Set("selection", selObj)

	// insert() method
	stateObj.Set("insert", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			state.Insert(call.Arguments[0].String())
		}
		return goja.Undefined()
	})

	// postError() method
	stateObj.Set("postError", func(call goja.FunctionCall) goja.Value {
		msg := ""
		if len(call.Arguments) > 0 {
			msg = call.Arguments[0].String()
		}
		state.PostError(msg)
		return goja.Undefined()
	})

	// postInfo() method
	stateObj.Set("postInfo", func(call goja.FunctionCall) goja.Value {
		msg := ""
		if len(call.Arguments) > 0 {
			msg = call.Arguments[0].String()
		}
		state.PostInfo(msg)
		return goja.Undefined()
	})

	vm.Set("state", stateObj)
	return nil
}

// registerBtoaAtob registers base64 encode/decode globals matching the browser API.
func registerBtoaAtob(vm *goja.Runtime) {
	vm.Set("btoa", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return vm.ToValue("")
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(call.Arguments[0].String()))
		return vm.ToValue(encoded)
	})

	vm.Set("atob", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return vm.ToValue("")
		}
		decoded, err := base64.StdEncoding.DecodeString(call.Arguments[0].String())
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("atob: %w", err)))
		}
		return vm.ToValue(string(decoded))
	})
}

// registerConsoleLog writes console.log calls to the application log file at INFO level.
func registerConsoleLog(vm *goja.Runtime, scriptName string) {
	console := vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, arg := range call.Arguments {
			parts[i] = arg.String()
		}
		logging.Log(logging.INFO, scriptName, strings.Join(parts, " "))
		return goja.Undefined()
	})
	vm.Set("console", console)
}
