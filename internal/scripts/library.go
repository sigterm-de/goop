package scripts

import (
	"sort"
	"strings"

	"github.com/sahilm/fuzzy"
)

// Library provides the combined searchable set of loaded scripts.
type Library interface {
	// All returns all scripts sorted by Bias ascending, then Source (BuiltIn
	// before UserProvided), then Name ascending (case-insensitive).
	All() []Script

	// Search returns scripts matching query using fuzzy matching on Name and Tags.
	// Returns All() when query is empty.
	// Results are sorted by match score descending; ties broken by All() order.
	Search(query string) []Script

	// Len returns the total number of loaded scripts.
	Len() int
}

// ScriptLibrary is the concrete implementation of Library.
type ScriptLibrary struct {
	sorted []Script // canonical sorted order (result of All())
}

// NewLibrary constructs a ScriptLibrary from a LoadResult and sorts scripts
// into canonical order immediately.
func NewLibrary(result LoadResult) *ScriptLibrary {
	scripts := make([]Script, len(result.Scripts))
	copy(scripts, result.Scripts)

	sort.SliceStable(scripts, func(i, j int) bool {
		a, b := scripts[i], scripts[j]
		if a.Bias != b.Bias {
			return a.Bias < b.Bias
		}
		if a.Source != b.Source {
			return a.Source < b.Source // BuiltIn (0) < UserProvided (1)
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})

	return &ScriptLibrary{sorted: scripts}
}

// All implements Library.
func (lib *ScriptLibrary) All() []Script {
	out := make([]Script, len(lib.sorted))
	copy(out, lib.sorted)
	return out
}

// Len implements Library.
func (lib *ScriptLibrary) Len() int {
	return len(lib.sorted)
}

// Search implements Library using sahilm/fuzzy.
func (lib *ScriptLibrary) Search(query string) []Script {
	if query == "" {
		return lib.All()
	}

	src := &scriptSource{scripts: lib.sorted}
	matches := fuzzy.FindFrom(query, src)

	if len(matches) == 0 {
		return []Script{} // never nil per contract TC-L-07
	}

	result := make([]Script, len(matches))
	for i, m := range matches {
		result[i] = lib.sorted[m.Index]
	}
	return result
}

// scriptSource implements fuzzy.Source so that fuzzy.FindFrom can search
// over script names.
type scriptSource struct {
	scripts []Script
}

func (s *scriptSource) String(i int) string { return s.scripts[i].Name }
func (s *scriptSource) Len() int            { return len(s.scripts) }
