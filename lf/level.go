package lf

import "strings"

// Level is the filtering intensity. lite = gentle trim, full = default,
// ultra = max compression. Mirrors lowfat-core/src/level.rs.
type Level int

const (
	Lite Level = iota
	Full
	Ultra
)

func (l Level) String() string {
	switch l {
	case Lite:
		return "lite"
	case Ultra:
		return "ultra"
	default:
		return "full"
	}
}

// HeadLimit scales a base head-limit by level: lite=2x, full=base,
// ultra=max(base/2, 5).
func (l Level) HeadLimit(base int) int {
	switch l {
	case Lite:
		return base * 2
	case Ultra:
		if h := base / 2; h > 5 {
			return h
		}
		return 5
	default:
		return base
	}
}

// ParseLevel parses "lite"/"full"/"ultra" (case-insensitive, matching lowfat's
// FromStr which lowercases first).
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "lite":
		return Lite, nil
	case "full":
		return Full, nil
	case "ultra":
		return Ultra, nil
	}
	return Full, &ParseError{"unknown level: " + s + " (expected: lite, full, ultra)"}
}

// ParseError is returned for malformed DSL input.
type ParseError struct{ Msg string }

func (e *ParseError) Error() string { return e.Msg }
