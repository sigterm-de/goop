package engine

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
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
	// Network, OS, and timer APIs are removed to prevent scripts from
	// communicating outside the sandbox.
	// eval() is removed because it executes strings with access to local scope,
	// enabling obfuscation techniques that could bypass future sandbox changes.
	// Function() is intentionally NOT removed: it only creates closures in the
	// global scope (no local variable access) and many legitimate Boop-compatible
	// libraries (e.g. node-forge) rely on it internally.
	poisoned := []string{
		"fetch", "XMLHttpRequest", "WebSocket",
		"process", "global", "Buffer",
		"setTimeout", "setInterval", "clearTimeout", "clearInterval",
		"eval",
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
	var timedOut atomic.Bool
	timer := time.AfterFunc(timeout, func() {
		timedOut.Store(true)
		vm.Interrupt(errTimeout)
	})
	defer timer.Stop()

	// ── Context cancellation ──────────────────────────────────────────────────
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			timedOut.Store(true)
			vm.Interrupt(errTimeout)
		case <-stop:
		}
	}()

	// ── Run: define functions + call main(state) ─────────────────────────────
	if _, runErr := vm.RunProgram(prog); runErr != nil {
		return e.runError(runErr, timedOut.Load(), timeout, input.ScriptName)
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
		return e.runError(callErr, timedOut.Load(), timeout, input.ScriptName)
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

	// selection — exposed as a frozen object so scripts cannot mutate it.
	// Mutations would be silently discarded by the Go side, which would
	// surprise script authors expecting Boop compatibility.
	selObj := vm.NewObject()
	if err := selObj.DefineDataProperty("start",
		vm.ToValue(state.SelectionInfo.Start),
		goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_TRUE,
	); err != nil {
		return fmt.Errorf("selection.start property: %w", err)
	}
	if err := selObj.DefineDataProperty("end",
		vm.ToValue(state.SelectionInfo.End),
		goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_TRUE,
	); err != nil {
		return fmt.Errorf("selection.end property: %w", err)
	}
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
//
// btoa() follows the spec: input must be a Latin-1 string (all code points ≤ 0xFF).
// Characters outside that range cause an InvalidCharacterError, matching Boop/browser
// behaviour. Callers that need to encode UTF-8 text should first apply
// encodeURIComponent (or the equivalent) to produce a Latin-1-safe string.
func registerBtoaAtob(vm *goja.Runtime) {
	vm.Set("btoa", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return vm.ToValue("")
		}
		s := call.Arguments[0].String()
		// Validate Latin-1: every code point must fit in one byte (≤ U+00FF).
		// Convert rune-by-rune so multi-byte UTF-8 sequences are handled correctly.
		runes := []rune(s)
		buf := make([]byte, len(runes))
		for i, r := range runes {
			if r > 0xFF {
				panic(vm.NewGoError(fmt.Errorf("InvalidCharacterError: btoa received a character (U+%04X) outside the Latin-1 range; encode to UTF-8 first with encodeURIComponent", r)))
			}
			buf[i] = byte(r)
		}
		return vm.ToValue(base64.StdEncoding.EncodeToString(buf))
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
