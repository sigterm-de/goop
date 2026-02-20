package engine

import (
	"fmt"
	"io/fs"
	"strings"

	"codeberg.org/daniel-ciaglia/goop/assets"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"gopkg.in/yaml.v3"
	"howett.net/plist"
)

// libFS holds the embedded scripts/lib directory for @boop/ JS modules.
var libFS fs.FS

func init() {
	sub, err := fs.Sub(assets.Scripts(), "lib")
	if err != nil {
		// lib dir missing — @boop/ JS modules will fail gracefully at require() time.
		return
	}
	libFS = sub
}

// registerModules sets up the @boop/ module namespace on the given require
// registry. All other require() paths will return "Cannot find module".
func registerModules(registry *require.Registry) {
	registry.RegisterNativeModule("@boop/yaml", yamlModuleLoader)
	registry.RegisterNativeModule("@boop/plist", plistModuleLoader)
}

// yamlModuleLoader exposes yaml.parse and yaml.stringify to scripts.
func yamlModuleLoader(runtime *goja.Runtime, module *goja.Object) {
	exports := module.Get("exports").(*goja.Object)

	exports.Set("parse", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(runtime.NewTypeError("yaml.parse requires a string argument"))
		}
		src := call.Arguments[0].String()
		var out any
		if err := yaml.Unmarshal([]byte(src), &out); err != nil {
			panic(runtime.NewGoError(fmt.Errorf("yaml.parse: %w", err)))
		}
		return runtime.ToValue(normaliseYAML(out))
	})

	exports.Set("stringify", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(runtime.NewTypeError("yaml.stringify requires an argument"))
		}
		val := call.Arguments[0].Export()
		b, err := yaml.Marshal(val)
		if err != nil {
			panic(runtime.NewGoError(fmt.Errorf("yaml.stringify: %w", err)))
		}
		return runtime.ToValue(string(b))
	})
}

// plistModuleLoader exposes plist.parse, plist.stringify, and plist.parseBinary.
func plistModuleLoader(runtime *goja.Runtime, module *goja.Object) {
	exports := module.Get("exports").(*goja.Object)

	exports.Set("parse", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(runtime.NewTypeError("plist.parse requires a string argument"))
		}
		src := call.Arguments[0].String()
		var out any
		if _, err := plist.Unmarshal([]byte(src), &out); err != nil {
			panic(runtime.NewGoError(fmt.Errorf("plist.parse: %w", err)))
		}
		return runtime.ToValue(out)
	})

	exports.Set("stringify", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(runtime.NewTypeError("plist.stringify requires an argument"))
		}
		val := call.Arguments[0].Export()
		b, err := plist.MarshalIndent(val, plist.XMLFormat, "\t")
		if err != nil {
			panic(runtime.NewGoError(fmt.Errorf("plist.stringify: %w", err)))
		}
		return runtime.ToValue(string(b))
	})

	exports.Set("parseBinary", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(runtime.NewTypeError("plist.parseBinary requires a string argument"))
		}
		src := call.Arguments[0].String()
		var out any
		if _, err := plist.Unmarshal([]byte(src), &out); err != nil {
			panic(runtime.NewGoError(fmt.Errorf("plist.parseBinary: %w", err)))
		}
		return runtime.ToValue(out)
	})
}

// normaliseYAML recursively converts map[interface{}]interface{} (produced by
// yaml.v3 when keys are not strings) to map[string]interface{} so that goja
// can expose it as a plain JS object.
func normaliseYAML(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[k] = normaliseYAML(vv)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[fmt.Sprintf("%v", k)] = normaliseYAML(vv)
		}
		return out
	case []any:
		for i, vv := range val {
			val[i] = normaliseYAML(vv)
		}
		return val
	default:
		return val
	}
}

// blockingRequireLoader serves @boop/ JS lib files from the embedded FS and
// rejects every other module path.
//
// goja_nodejs normalises require("@boop/foo") to the path
// "node_modules/@boop/foo" before calling the loader, so we handle both
// forms.
func blockingRequireLoader(path string) ([]byte, error) {
	// Strip node_modules/ prefix added by goja_nodejs resolver.
	modPath := strings.TrimPrefix(path, "node_modules/")

	if after, ok := strings.CutPrefix(modPath, "@boop/"); ok {
		name := after
		if libFS != nil {
			data, err := fs.ReadFile(libFS, name+".js")
			if err == nil {
				return data, nil
			}
		}
		// Native module already registered — signal not-found so the registry
		// uses the native loader instead.
		return nil, require.ModuleFileDoesNotExistError
	}
	return nil, fmt.Errorf("cannot find module '%s'", path)
}
