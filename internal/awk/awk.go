// Package awk provides small pure-Go primitives that mirror the awk/sed idioms
// used by lowfat's bundled filters, so filter ShellRunners can reimplement their
// shell: ops without spawning a subprocess. These match awk's whitespace field
// splitting and POSIX sed line semantics for ASCII tool output.
package awk

import (
	"regexp"
	"strings"
)

// Lines splits text the way the lowfat engine does (Rust str::lines()): on '\n',
// dropping a trailing empty from a final newline, stripping a trailing '\r'.
func Lines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	for i := range parts {
		parts[i] = strings.TrimSuffix(parts[i], "\r")
	}
	return parts
}

// Fields splits on runs of whitespace, dropping leading/trailing — exactly like
// awk's default field splitting. Returns nil for a blank line.
func Fields(line string) []string { return strings.Fields(line) }

// Field returns awk's $i (1-based). $0 returns the whole line. Out-of-range
// positive indices return "" (awk yields an empty string for unset fields).
func Field(line string, i int) string {
	if i == 0 {
		return line
	}
	f := strings.Fields(line)
	if i < 1 || i > len(f) {
		return ""
	}
	return f[i-1]
}

var multiSpace = regexp.MustCompile(` +`)

// CollapseSpaces mirrors `sed 's/  */ /g'`: collapse each run of one-or-more
// spaces to a single space, per line.
func CollapseSpaces(s string) string {
	lines := Lines(s)
	for i, l := range lines {
		lines[i] = multiSpace.ReplaceAllString(l, " ")
	}
	return strings.Join(lines, "\n")
}

// MapLines applies fn to each line and rejoins with '\n'.
func MapLines(s string, fn func(string) string) string {
	lines := Lines(s)
	for i, l := range lines {
		lines[i] = fn(l)
	}
	return strings.Join(lines, "\n")
}

// KeepLines keeps lines for which pred is true.
func KeepLines(s string, pred func(string) bool) string {
	var out []string
	for _, l := range Lines(s) {
		if pred(l) {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

// Head returns the first n lines.
func Head(s string, n int) string {
	lines := Lines(s)
	if n < len(lines) {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
