// Package lf is a pure-Go port of lowfat's .lf filter DSL: a line-oriented,
// indentation-sensitive language for compacting command output. It contains the
// parser (Parse) and executor (Execute). Filter-specific shell: / or-shell: ops
// (awk/sed) are delegated to a ShellRunner supplied by the caller, so the engine
// itself spawns no subprocesses.
package lf

import "regexp"

// RuleSet is a parsed .lf file: macro definitions plus selector rules.
type RuleSet struct {
	Defines []Define
	Rules   []Rule
}

func (rs *RuleSet) findDefine(name string) *Define {
	for i := range rs.Defines {
		if rs.Defines[i].Name == name {
			return &rs.Defines[i]
		}
	}
	return nil
}

// selectRule returns the first rule matching (sub, level); first-match-wins.
func (rs *RuleSet) selectRule(sub string, level Level) *Rule {
	for i := range rs.Rules {
		if rs.Rules[i].matches(sub, level) {
			return &rs.Rules[i]
		}
	}
	return nil
}

type Define struct {
	Name   string
	Params []string
	Ops    []Op
}

type Rule struct {
	Sub    SubPattern
	Level  LevelPattern
	Ops    []Op
	LineNo int
}

func (r *Rule) matches(sub string, level Level) bool {
	subOK := r.Sub.Star
	if !subOK {
		for _, a := range r.Sub.Alt {
			if globMatch(a, sub) {
				subOK = true
				break
			}
		}
	}
	lvlOK := r.Level.Star || r.Level.Level == level
	return subOK && lvlOK
}

type SubPattern struct {
	Star bool
	Alt  []string
}

type LevelPattern struct {
	Star  bool
	Level Level
}

type OpKind int

const (
	OpKeep OpKind = iota
	OpDrop
	OpHead
	OpTail
	OpOr
	OpOrShell
	OpRaw
	OpCascade
	OpShell
	OpPython
	OpMacroCall
	OpSplit
)

type Op struct {
	Kind OpKind

	Pattern *regexp.Regexp // Keep/Drop/Split delimiter
	PatSrc  string

	HeadAuto bool // Head/Tail
	HeadN    int

	OrText string // Or

	Body string // OrShell/Shell/Python

	Branches []Branch // Cascade

	MacroName string // MacroCall
	MacroArgs []MacroArg

	Pre  []Op // Split
	Post []Op
}

// Branch is one arm of a cascade. Guard == nil means the else arm.
type Branch struct {
	Guard *Guard
	Ops   []Op
}

// Guard is an AND of atoms.
type Guard struct {
	Atoms []Atom
}

type AtomKind int

const (
	AtomExit AtomKind = iota
	AtomLevel
	AtomFlag
)

type Atom struct {
	Kind  AtomKind
	Exit  ExitMatch
	Level Level
	Flag  string
}

type ExitMatch int

const (
	ExitOk ExitMatch = iota
	ExitFailed
)

type MacroArg struct {
	IsNumber bool
	Number   int
	Str      string
}
