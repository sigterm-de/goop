package scripts

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ScriptSource distinguishes embedded (built-in) scripts from user-provided ones.
type ScriptSource int

const (
	BuiltIn      ScriptSource = iota // Embedded via go:embed at compile time
	UserProvided                     // Loaded from user config directory at startup
)

// Script holds the parsed metadata and full source of a single Boop script.
type Script struct {
	Name        string
	Description string
	Icon        string   // FontAwesome HTML or icon name; empty if not declared
	Tags        []string // Empty slice if not declared
	Bias        float64  // Default 0.0 — lower values sort earlier
	Source      ScriptSource
	FilePath    string // Virtual path for built-ins; absolute path for user scripts
	Content     string // Full JavaScript source (including header)
}

// errNoHeader is returned when the file does not start with /**!.
var errNoHeader = errors.New("missing /**! header")

// ParseHeader parses the /**! metadata header from a Boop script source and
// returns a Script with all fields populated. Content is always set to the
// full source text regardless of parse success.
//
// Returns an error when:
//   - The content does not start with "/**!" (errNoHeader)
//   - @name or @description are missing or empty after trimming
func ParseHeader(content string) (Script, error) {
	s := Script{
		Content: content,
		Tags:    []string{},
	}

	// Strip UTF-8 BOM if present
	body := strings.TrimPrefix(content, "\xef\xbb\xbf")

	if !strings.HasPrefix(body, "/**!") {
		return s, errNoHeader
	}

	// Find the closing */ of the header block
	end := strings.Index(body, "*/")
	if end < 0 {
		return s, fmt.Errorf("unclosed /**! header block")
	}
	headerBlock := body[4:end] // everything between /**! and */

	for line := range strings.SplitSeq(headerBlock, "\n") {
		// Strip leading whitespace and optional leading '*'
		trimmed := strings.TrimLeft(line, " \t")
		trimmed = strings.TrimPrefix(trimmed, "*")
		trimmed = strings.TrimLeft(trimmed, " \t")

		if !strings.HasPrefix(trimmed, "@") {
			continue
		}

		// Split on first whitespace to separate key from value
		idx := strings.IndexAny(trimmed, " \t")
		if idx < 0 {
			continue
		}
		key := trimmed[1:idx] // strip leading '@'
		val := strings.TrimSpace(trimmed[idx+1:])

		switch key {
		case "name":
			s.Name = val
		case "description":
			s.Description = val
		case "icon":
			s.Icon = val
		case "tags":
			for tag := range strings.SplitSeq(val, ",") {
				if t := strings.TrimSpace(tag); t != "" {
					s.Tags = append(s.Tags, t)
				}
			}
		case "bias":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				s.Bias = f
			}
			// Unknown keys are silently ignored
		}
	}

	if strings.TrimSpace(s.Name) == "" {
		return s, fmt.Errorf("/**! header missing @name")
	}
	if strings.TrimSpace(s.Description) == "" {
		return s, fmt.Errorf("/**! header missing @description")
	}

	return s, nil
}
