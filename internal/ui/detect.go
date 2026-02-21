package ui

import (
	"encoding/json"
	"encoding/xml"
	"strings"

	"gopkg.in/yaml.v3"
)

// maxDetectBytes is the content-size limit beyond which detection is skipped.
// Analysing multi-megabyte content is unlikely to be useful and could exceed
// the 50 ms performance budget.
const maxDetectBytes = 4 * 1024 * 1024 // 4 MB

// Detect returns the GtkSourceView language ID and display name for the given
// content using a two-tier approach: a cheap structural heuristic acts as the
// first gate, and a stdlib parse validates the candidate before the result is
// accepted. Returns ("", "") when no format can be identified with confidence.
//
// Supported auto-detection: JSON, HTML, XML, YAML.
// SQL and Markdown are excluded — they lack reliable detection heuristics that
// satisfy the zero-false-positive requirement.
func Detect(content string) (langID, langName string) {
	content = strings.TrimSpace(content)
	if content == "" || len(content) > maxDetectBytes {
		return "", ""
	}

	// HTML is tested before XML because HTML can look like malformed XML.
	switch {
	case isHTML(content):
		return "html", "HTML"
	case isJSON(content):
		return "json", "JSON"
	case isXML(content):
		return "xml", "XML"
	case isYAML(content):
		return "yaml", "YAML"
	}
	return "", ""
}

// isJSON returns true when content starts with { or [ (heuristic) and passes
// json.Valid (validation).
func isJSON(s string) bool {
	if s[0] != '{' && s[0] != '[' {
		return false
	}
	return json.Valid([]byte(s))
}

// isHTML returns true when the content contains an HTML doctype or opening
// html element in its first 512 bytes. HTML5 is not strict XML, so we treat
// the heuristic match itself as sufficient validation.
func isHTML(s string) bool {
	prefix := strings.ToLower(s[:min(512, len(s))])
	return strings.Contains(prefix, "<!doctype html") ||
		strings.Contains(prefix, "<html")
}

// isXML returns true when content starts with an XML declaration or an
// element, and the stdlib XML decoder can advance past at least one token.
func isXML(s string) bool {
	if !strings.HasPrefix(s, "<?xml") {
		// Must start with '<' followed by a letter or '!' (element or comment).
		if len(s) < 2 || s[0] != '<' {
			return false
		}
		second := s[1]
		if !isLetter(second) && second != '!' {
			return false
		}
	}
	d := xml.NewDecoder(strings.NewReader(s))
	_, err := d.Token()
	return err == nil
}

// isYAML returns true when content starts with a YAML document separator or
// its first non-empty line looks like a YAML mapping key (with or without a
// value on the same line), and yaml.Unmarshal succeeds.
func isYAML(s string) bool {
	if !strings.HasPrefix(s, "---") {
		if !looksLikeYAMLLine(firstNonEmptyLine(s)) {
			return false
		}
	}
	var v any
	return yaml.Unmarshal([]byte(s), &v) == nil && v != nil
}

// looksLikeYAMLLine returns true when line is a plausible YAML mapping entry:
// a simple key (no internal spaces or slashes) followed by ':' at end of line
// or by ': ' / ':\t' (key-value pair).
func looksLikeYAMLLine(line string) bool {
	idx := strings.IndexByte(line, ':')
	if idx < 1 {
		return false
	}
	key := line[:idx]
	// Key must be a simple identifier — no spaces, tabs, or slashes.
	if strings.ContainsAny(key, " \t/") {
		return false
	}
	rest := line[idx+1:]
	// After colon: end of line (bare key), space (key: value), or tab.
	return rest == "" || rest[0] == ' ' || rest[0] == '\t'
}

// firstNonEmptyLine returns the first non-blank line from s, scanning at most
// the first ten lines.
func firstNonEmptyLine(s string) string {
	for i, line := range strings.SplitN(s, "\n", 11) {
		if i == 10 {
			break
		}
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// isLetter reports whether b is an ASCII letter.
func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
