package lf

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// line mirrors the Rust `Line` preprocessing struct.
type line struct {
	indent int
	text   string // trimmed both ends; "" if blank
	raw    string // original, no trailing newline
	lineNo int
	isMeta bool // blank or top-level comment
}

func splitLines(input string) []line {
	parts := strings.Split(input, "\n")
	out := make([]line, 0, len(parts))
	for i, rawLine := range parts {
		raw := strings.TrimRight(rawLine, "\r")
		stripped := strings.TrimLeftFunc(raw, unicode.IsSpace)
		indent := len(raw) - len(stripped)
		text := strings.TrimRightFunc(stripped, unicode.IsSpace)
		isMeta := text == "" || strings.HasPrefix(text, "#")
		out = append(out, line{indent, text, raw, i + 1, isMeta})
	}
	return out
}

var opKeywords = map[string]bool{
	"keep": true, "drop": true, "head": true, "tail": true, "or": true,
	"or-shell:": true, "else": true, "else-shell:": true, "shell:": true,
	"python:": true, "split": true, "raw": true, "passthrough": true,
	"if": true, "elif": true, "match": true,
}

func Parse(input string) (*RuleSet, error) {
	lines := splitLines(input)
	p := &parser{lines: lines, macroNames: collectMacroNames(lines)}
	return p.parseRuleSet()
}

func collectMacroNames(lines []line) []string {
	var names []string
	for _, l := range lines {
		if l.isMeta {
			continue
		}
		if rest, ok := strings.CutPrefix(l.text, "define "); ok {
			end := strings.IndexFunc(rest, func(c rune) bool {
				return c == '(' || c == ':' || unicode.IsSpace(c)
			})
			if end < 0 {
				end = len(rest)
			}
			name := strings.TrimSpace(rest[:end])
			if name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}

type parser struct {
	lines      []line
	pos        int
	macroNames []string
}

func (p *parser) peekSignificant() *line {
	for p.pos < len(p.lines) {
		if p.lines[p.pos].isMeta {
			p.pos++
		} else {
			return &p.lines[p.pos]
		}
	}
	return nil
}

func (p *parser) advance() *line {
	if p.pos < len(p.lines) {
		l := &p.lines[p.pos]
		p.pos++
		return l
	}
	return nil
}

func (p *parser) isMacro(name string) bool {
	for _, n := range p.macroNames {
		if n == name {
			return true
		}
	}
	return false
}

func (p *parser) parseRuleSet() (*RuleSet, error) {
	rs := &RuleSet{}
	for {
		l := p.peekSignificant()
		if l == nil {
			break
		}
		if l.indent != 0 {
			return nil, fmt.Errorf("line %d: unexpected indent at top level", l.lineNo)
		}
		if strings.HasPrefix(l.text, "define ") {
			d, err := p.parseDefine()
			if err != nil {
				return nil, err
			}
			rs.Defines = append(rs.Defines, d)
		} else {
			r, err := p.parseRule()
			if err != nil {
				return nil, err
			}
			rs.Rules = append(rs.Rules, r)
		}
	}
	return rs, nil
}

func (p *parser) parseDefine() (Define, error) {
	header := p.advance()
	lineNo := header.lineNo
	rest := strings.TrimPrefix(header.text, "define ")
	name, params, afterParen, err := parseDefineHeader(rest)
	if err != nil {
		return Define{}, fmt.Errorf("line %d: %w", lineNo, err)
	}
	if !strings.HasPrefix(afterParen, ":") {
		return Define{}, fmt.Errorf("line %d: expected `:` after define header, got `%s`", lineNo, afterParen)
	}
	trailing := strings.TrimSpace(afterParen[1:])
	if trailing != "" {
		return Define{}, fmt.Errorf("line %d: one-line `define` body not supported (use indented body)", lineNo)
	}
	ops, err := p.parseIndentedOps(header.indent)
	if err != nil {
		return Define{}, err
	}
	if len(ops) == 0 {
		return Define{}, fmt.Errorf("line %d: `define %s` has empty body", lineNo, name)
	}
	return Define{Name: name, Params: params, Ops: ops}, nil
}

func (p *parser) parseRule() (Rule, error) {
	header := p.advance()
	lineNo := header.lineNo
	parentIndent := header.indent
	colon := strings.IndexByte(header.text, ':')
	if colon < 0 {
		return Rule{}, fmt.Errorf("line %d: missing `:` in rule header", lineNo)
	}
	selector := header.text[:colon]
	after := header.text[colon+1:]
	sub, level, err := parseSelector(selector)
	if err != nil {
		return Rule{}, fmt.Errorf("line %d: %w", lineNo, err)
	}

	var ops []Op
	inline := strings.TrimSpace(after)
	if inline != "" {
		io, err := p.parseInlineOps(inline, lineNo)
		if err != nil {
			return Rule{}, err
		}
		ops = append(ops, io...)
		more, err := p.parseIndentedOps(parentIndent)
		if err != nil {
			return Rule{}, err
		}
		ops = append(ops, more...)
	} else {
		ops, err = p.parseBody(parentIndent)
		if err != nil {
			return Rule{}, err
		}
	}
	if len(ops) == 0 {
		return Rule{}, fmt.Errorf("line %d: rule has no ops", lineNo)
	}
	return Rule{Sub: sub, Level: level, Ops: ops, LineNo: lineNo}, nil
}

func (p *parser) parseIndentedOps(parentIndent int) ([]Op, error) {
	var ops []Op
	for {
		l := p.peekSignificant()
		if l == nil || l.indent <= parentIndent {
			break
		}
		op, err := p.parseOpLine()
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, nil
}

func (p *parser) parseBody(parentIndent int) ([]Op, error) {
	if l := p.peekSignificant(); l != nil && l.indent > parentIndent {
		if isBodyOpener(l.text, "if") {
			br, err := p.parseCascade(parentIndent)
			if err != nil {
				return nil, err
			}
			return []Op{{Kind: OpCascade, Branches: br}}, nil
		}
		if isBodyOpener(l.text, "match") {
			br, err := p.parseMatch(parentIndent)
			if err != nil {
				return nil, err
			}
			return []Op{{Kind: OpCascade, Branches: br}}, nil
		}
	}
	return p.parseIndentedOps(parentIndent)
}

func (p *parser) parseCascade(parentIndent int) ([]Branch, error) {
	var branches []Branch
	armIndent := -1
	for {
		l := p.peekSignificant()
		if l == nil || l.indent <= parentIndent {
			break
		}
		if armIndent == -1 {
			armIndent = l.indent
		} else if l.indent != armIndent {
			break
		}
		lineNo := l.lineNo
		kw := leadingAlpha(l.text)
		switch {
		case kw == "if" && len(branches) == 0:
		case (kw == "elif" || kw == "else") && len(branches) > 0:
		case kw == "if":
			return nil, fmt.Errorf("line %d: unexpected `if` — cascade already open", lineNo)
		case kw == "elif" || kw == "else":
			return nil, fmt.Errorf("line %d: `%s` without a leading `if`", lineNo, kw)
		default:
			return branches, nil
		}
		br, err := p.parseBranch(kw)
		if err != nil {
			return nil, err
		}
		isElse := br.Guard == nil
		branches = append(branches, br)
		if isElse {
			break
		}
	}
	return branches, nil
}

func (p *parser) parseBranch(head string) (Branch, error) {
	l := p.advance()
	lineNo := l.lineNo
	indent := l.indent
	rest := strings.TrimLeft(l.text[len(head):], " \t")
	colon := strings.IndexByte(rest, ':')
	if colon < 0 {
		return Branch{}, fmt.Errorf("line %d: missing `:` in `%s` arm", lineNo, head)
	}
	guardStr := strings.TrimSpace(rest[:colon])
	after := strings.TrimSpace(rest[colon+1:])
	var guard *Guard
	if head == "else" {
		if guardStr != "" {
			return Branch{}, fmt.Errorf("line %d: `else` takes no guard", lineNo)
		}
	} else {
		g, err := parseGuard(guardStr, lineNo)
		if err != nil {
			return Branch{}, err
		}
		guard = g
	}
	ops, err := p.parseArmBody(after, indent, lineNo)
	if err != nil {
		return Branch{}, err
	}
	if len(ops) == 0 {
		return Branch{}, fmt.Errorf("line %d: `%s` arm has no ops", lineNo, head)
	}
	return Branch{Guard: guard, Ops: ops}, nil
}

func (p *parser) parseArmBody(inline string, indent, lineNo int) ([]Op, error) {
	var ops []Op
	if inline != "" {
		io, err := p.parseInlineOps(inline, lineNo)
		if err != nil {
			return nil, err
		}
		ops = append(ops, io...)
	}
	if len(ops) == 0 {
		if child := p.peekSignificant(); child != nil && child.indent > indent {
			if isBodyOpener(child.text, "if") {
				br, err := p.parseCascade(indent)
				if err != nil {
					return nil, err
				}
				return []Op{{Kind: OpCascade, Branches: br}}, nil
			}
			if isBodyOpener(child.text, "match") {
				br, err := p.parseMatch(indent)
				if err != nil {
					return nil, err
				}
				return []Op{{Kind: OpCascade, Branches: br}}, nil
			}
		}
	}
	more, err := p.parseIndentedOps(indent)
	if err != nil {
		return nil, err
	}
	ops = append(ops, more...)
	return ops, nil
}

type matchDim int

const (
	dimLevel matchDim = iota
	dimExit
)

func (p *parser) parseMatch(parentIndent int) ([]Branch, error) {
	header := p.advance()
	lineNo := header.lineNo
	rest := strings.TrimLeft(strings.TrimPrefix(header.text, "match"), " \t")
	colon := strings.IndexByte(rest, ':')
	if colon < 0 {
		return nil, fmt.Errorf("line %d: missing `:` after match dimension", lineNo)
	}
	dimStr := strings.TrimSpace(rest[:colon])
	trailing := strings.TrimSpace(rest[colon+1:])
	if trailing != "" {
		return nil, fmt.Errorf("line %d: `match` header doesn't take inline ops (got `%s`)", lineNo, trailing)
	}
	dim, err := parseMatchDim(dimStr, lineNo)
	if err != nil {
		return nil, err
	}

	var branches []Branch
	armIndent := -1
	for {
		l := p.peekSignificant()
		if l == nil || l.indent <= parentIndent {
			break
		}
		if armIndent == -1 {
			armIndent = l.indent
		} else if l.indent != armIndent {
			break
		}
		br, err := p.parseMatchArm(dim)
		if err != nil {
			return nil, err
		}
		isElse := br.Guard == nil
		branches = append(branches, br)
		if isElse {
			break
		}
	}
	if len(branches) == 0 {
		return nil, fmt.Errorf("line %d: `match` has no arms", lineNo)
	}
	return branches, nil
}

func (p *parser) parseMatchArm(dim matchDim) (Branch, error) {
	l := p.advance()
	lineNo := l.lineNo
	indent := l.indent
	colon := strings.IndexByte(l.text, ':')
	if colon < 0 {
		return Branch{}, fmt.Errorf("line %d: missing `:` in match arm", lineNo)
	}
	value := strings.TrimSpace(l.text[:colon])
	after := strings.TrimSpace(l.text[colon+1:])

	var guard *Guard
	if value != "else" {
		atom, err := buildMatchAtom(dim, value, lineNo)
		if err != nil {
			return Branch{}, err
		}
		guard = &Guard{Atoms: []Atom{atom}}
	}
	ops, err := p.parseArmBody(after, indent, lineNo)
	if err != nil {
		return Branch{}, err
	}
	if len(ops) == 0 {
		return Branch{}, fmt.Errorf("line %d: match arm `%s` has no ops", lineNo, value)
	}
	return Branch{Guard: guard, Ops: ops}, nil
}

func (p *parser) parseOpLine() (Op, error) {
	l := p.advance()
	lineNo := l.lineNo
	indent := l.indent
	text := l.text
	head, _ := splitFirstWord(text)

	switch {
	case head == "keep":
		re, err := parseRegexLiteral(strings.TrimLeft(text[len(head):], " \t"), lineNo)
		if err != nil {
			return Op{}, err
		}
		return Op{Kind: OpKeep, Pattern: re.re, PatSrc: re.src}, nil
	case head == "drop":
		re, err := parseRegexLiteral(strings.TrimLeft(text[len(head):], " \t"), lineNo)
		if err != nil {
			return Op{}, err
		}
		return Op{Kind: OpDrop, Pattern: re.re, PatSrc: re.src}, nil
	case head == "head":
		auto, n, err := parseHeadArg(strings.TrimSpace(text[len(head):]), lineNo)
		if err != nil {
			return Op{}, err
		}
		return Op{Kind: OpHead, HeadAuto: auto, HeadN: n}, nil
	case head == "tail":
		auto, n, err := parseHeadArg(strings.TrimSpace(text[len(head):]), lineNo)
		if err != nil {
			return Op{}, err
		}
		return Op{Kind: OpTail, HeadAuto: auto, HeadN: n}, nil
	case head == "or" || head == "else":
		s, err := parseStringLiteral(strings.TrimLeft(text[len(head):], " \t"), lineNo)
		if err != nil {
			return Op{}, err
		}
		return Op{Kind: OpOr, OrText: s}, nil
	case head == "or-shell:" || head == "else-shell:":
		body := strings.TrimLeft(text[len(head):], " \t")
		if body == "" {
			return Op{}, fmt.Errorf("line %d: `%s` requires a command", lineNo, head)
		}
		return Op{Kind: OpOrShell, Body: body}, nil
	case head == "raw" || head == "passthrough":
		return Op{Kind: OpRaw}, nil
	case head == "shell:":
		body, err := p.parseBlockBody(text, head, indent, lineNo)
		if err != nil {
			return Op{}, err
		}
		return Op{Kind: OpShell, Body: body}, nil
	case head == "python:":
		body, err := p.parseBlockBody(text, head, indent, lineNo)
		if err != nil {
			return Op{}, err
		}
		return Op{Kind: OpPython, Body: body}, nil
	case head == "split":
		re, err := parseRegexLiteral(strings.TrimLeft(text[len(head):], " \t"), lineNo)
		if err != nil {
			return Op{}, err
		}
		pre, post, err := p.parseSplitBranches(indent)
		if err != nil {
			return Op{}, err
		}
		if len(pre) == 0 && len(post) == 0 {
			return Op{}, fmt.Errorf("line %d: `split` needs at least one `pre:` or `post:` block", lineNo)
		}
		return Op{Kind: OpSplit, Pattern: re.re, PatSrc: re.src, Pre: pre, Post: post}, nil
	case p.isMacro(head):
		args, err := parseMacroArgs(strings.TrimSpace(text[len(head):]), lineNo)
		if err != nil {
			return Op{}, err
		}
		return Op{Kind: OpMacroCall, MacroName: head, MacroArgs: args}, nil
	}
	return Op{}, fmt.Errorf("line %d: unknown op `%s`", lineNo, head)
}

func (p *parser) parseBlockBody(lineText, head string, parentIndent, lineNo int) (string, error) {
	after := strings.TrimLeft(lineText[len(head):], " \t")
	if after != "|" {
		if after == "" {
			return "", fmt.Errorf("line %d: empty `%s` body (use `| <newline>` for block form)", lineNo, head)
		}
		return after, nil
	}
	var collected []*line
	base := -1
	for p.pos < len(p.lines) {
		l := &p.lines[p.pos]
		if l.text == "" {
			collected = append(collected, l)
			p.pos++
			continue
		}
		if l.indent <= parentIndent {
			break
		}
		if base == -1 {
			base = l.indent
		}
		collected = append(collected, l)
		p.pos++
	}
	for len(collected) > 0 && collected[len(collected)-1].text == "" {
		collected = collected[:len(collected)-1]
	}
	if len(collected) == 0 {
		return "", fmt.Errorf("line %d: `%s` block is empty", lineNo, head)
	}
	if base == -1 {
		base = parentIndent + 4
	}
	dedented := make([]string, len(collected))
	for i, l := range collected {
		switch {
		case l.text == "":
			dedented[i] = ""
		case len(l.raw) >= base:
			dedented[i] = l.raw[base:]
		default:
			dedented[i] = strings.TrimLeft(l.raw, " \t")
		}
	}
	return strings.Join(dedented, "\n"), nil
}

func (p *parser) parseSplitBranches(parentIndent int) ([]Op, []Op, error) {
	var pre, post []Op
	for {
		l := p.peekSignificant()
		if l == nil || l.indent != parentIndent {
			break
		}
		switch l.text {
		case "pre:":
			p.advance()
			ops, err := p.parseIndentedOps(parentIndent)
			if err != nil {
				return nil, nil, err
			}
			pre = ops
		case "post:":
			p.advance()
			ops, err := p.parseIndentedOps(parentIndent)
			if err != nil {
				return nil, nil, err
			}
			post = ops
		default:
			return pre, post, nil
		}
	}
	return pre, post, nil
}

func (p *parser) parseInlineOps(text string, lineNo int) ([]Op, error) {
	var ops []Op
	remaining := strings.TrimSpace(text)
	for remaining != "" {
		head, _ := splitFirstWord(remaining)
		switch {
		case head == "shell:":
			body := strings.TrimLeft(remaining[len(head):], " \t")
			if body == "" {
				return nil, fmt.Errorf("line %d: inline `shell:` needs a command", lineNo)
			}
			ops = append(ops, Op{Kind: OpShell, Body: body})
			remaining = ""
		case head == "python:":
			body := strings.TrimLeft(remaining[len(head):], " \t")
			if body == "" {
				return nil, fmt.Errorf("line %d: inline `python:` needs a command", lineNo)
			}
			ops = append(ops, Op{Kind: OpPython, Body: body})
			remaining = ""
		case head == "or-shell:" || head == "else-shell:":
			body := strings.TrimLeft(remaining[len(head):], " \t")
			if body == "" {
				return nil, fmt.Errorf("line %d: inline `%s` needs a command", lineNo, head)
			}
			ops = append(ops, Op{Kind: OpOrShell, Body: body})
			remaining = ""
		case head == "raw" || head == "passthrough":
			ops = append(ops, Op{Kind: OpRaw})
			remaining = strings.TrimLeft(remaining[len(head):], " \t")
		case head == "keep" || head == "drop":
			rest := strings.TrimLeft(remaining[len(head):], " \t")
			re, after, err := parseRegexLiteralAndRest(rest, lineNo)
			if err != nil {
				return nil, err
			}
			kind := OpKeep
			if head == "drop" {
				kind = OpDrop
			}
			ops = append(ops, Op{Kind: kind, Pattern: re.re, PatSrc: re.src})
			remaining = strings.TrimLeft(after, " \t")
		case head == "head" || head == "tail":
			rest := strings.TrimLeft(remaining[len(head):], " \t")
			word, after := takeWord(rest)
			auto, n, err := parseHeadArg(word, lineNo)
			if err != nil {
				return nil, err
			}
			kind := OpHead
			if head == "tail" {
				kind = OpTail
			}
			ops = append(ops, Op{Kind: kind, HeadAuto: auto, HeadN: n})
			remaining = strings.TrimLeft(after, " \t")
		case head == "or" || head == "else":
			rest := strings.TrimLeft(remaining[len(head):], " \t")
			s, after, err := parseStringLiteralAndRest(rest, lineNo)
			if err != nil {
				return nil, err
			}
			ops = append(ops, Op{Kind: OpOr, OrText: s})
			remaining = strings.TrimLeft(after, " \t")
		case head == "split":
			return nil, fmt.Errorf("line %d: `split` cannot appear inline (needs pre:/post: blocks)", lineNo)
		case p.isMacro(head):
			rest := strings.TrimLeft(remaining[len(head):], " \t")
			args, after, err := parseMacroArgsUntilOp(rest, p.macroNames, lineNo)
			if err != nil {
				return nil, err
			}
			ops = append(ops, Op{Kind: OpMacroCall, MacroName: head, MacroArgs: args})
			remaining = strings.TrimLeft(after, " \t")
		default:
			return nil, fmt.Errorf("line %d: unknown op `%s` in inline chain", lineNo, head)
		}
	}
	return ops, nil
}

// ── free-function sub-parsers ──────────────────────────────────────

func isBodyOpener(text, kw string) bool {
	rest, ok := strings.CutPrefix(text, kw)
	if !ok {
		return false
	}
	if rest == "" {
		return true
	}
	r := rune(rest[0])
	return unicode.IsSpace(r) || r == ':'
}

func splitFirstWord(s string) (string, string) {
	s = strings.TrimLeft(s, " \t")
	end := strings.IndexFunc(s, unicode.IsSpace)
	if end < 0 {
		end = len(s)
	}
	return s[:end], s[end:]
}

func takeWord(s string) (string, string) { return splitFirstWord(s) }

func leadingAlpha(s string) string {
	end := 0
	for end < len(s) && ((s[end] >= 'a' && s[end] <= 'z') || (s[end] >= 'A' && s[end] <= 'Z')) {
		end++
	}
	return s[:end]
}

func parseSelector(s string) (SubPattern, LevelPattern, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return SubPattern{}, LevelPattern{}, fmt.Errorf("empty selector")
	}
	parts := strings.SplitN(s, ",", 2)
	subStr := strings.TrimSpace(parts[0])
	levelStr := "*"
	if len(parts) > 1 {
		levelStr = strings.TrimSpace(parts[1])
	}
	var sub SubPattern
	if subStr == "*" {
		sub = SubPattern{Star: true}
	} else {
		alts := strings.Split(subStr, "|")
		for i := range alts {
			alts[i] = strings.TrimSpace(alts[i])
			if alts[i] == "" {
				return SubPattern{}, LevelPattern{}, fmt.Errorf("empty alternative in sub pattern `%s`", subStr)
			}
		}
		sub = SubPattern{Alt: alts}
	}
	var level LevelPattern
	if levelStr == "*" {
		level = LevelPattern{Star: true}
	} else {
		lvl, err := ParseLevel(levelStr)
		if err != nil {
			return SubPattern{}, LevelPattern{}, err
		}
		level = LevelPattern{Level: lvl}
	}
	return sub, level, nil
}

func globMatch(pat, text string) bool {
	star := strings.IndexByte(pat, '*')
	if star < 0 {
		return pat == text
	}
	prefix := pat[:star]
	rest := pat[star+1:]
	tail, ok := strings.CutPrefix(text, prefix)
	if !ok {
		return false
	}
	if rest == "" {
		return true
	}
	for i := 0; i <= len(tail); i++ {
		if globMatch(rest, tail[i:]) {
			return true
		}
	}
	return false
}

func parseGuard(s string, lineNo int) (*Guard, error) {
	var atoms []Atom
	for _, part := range strings.Split(s, " and ") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("line %d: empty guard", lineNo)
		}
		a, err := parseAtom(part, lineNo)
		if err != nil {
			return nil, err
		}
		atoms = append(atoms, a)
	}
	if len(atoms) == 0 {
		return nil, fmt.Errorf("line %d: empty guard", lineNo)
	}
	return &Guard{Atoms: atoms}, nil
}

func parseAtom(s string, lineNo int) (Atom, error) {
	if strings.HasPrefix(s, "-") {
		return Atom{Kind: AtomFlag, Flag: s}, nil
	}
	words := strings.Fields(s)
	dim := ""
	if len(words) > 0 {
		dim = words[0]
	}
	var val string
	hasVal := len(words) > 1
	if hasVal {
		val = words[1]
	}
	if len(words) > 2 {
		return Atom{}, fmt.Errorf("line %d: guard `%s` has too many words", lineNo, s)
	}
	switch dim {
	case "exit":
		if !hasVal {
			return Atom{}, fmt.Errorf("line %d: `exit` guard needs a value (ok|failed)", lineNo)
		}
		switch val {
		case "ok":
			return Atom{Kind: AtomExit, Exit: ExitOk}, nil
		case "failed":
			return Atom{Kind: AtomExit, Exit: ExitFailed}, nil
		default:
			return Atom{}, fmt.Errorf("line %d: unknown exit value `%s` (expected ok|failed)", lineNo, val)
		}
	case "level":
		if !hasVal {
			return Atom{}, fmt.Errorf("line %d: `level` guard needs a value", lineNo)
		}
		lvl, err := ParseLevel(val)
		if err != nil {
			return Atom{}, fmt.Errorf("line %d: %w", lineNo, err)
		}
		return Atom{Kind: AtomLevel, Level: lvl}, nil
	default:
		return Atom{}, fmt.Errorf("line %d: unknown guard `%s` (expected `exit ...`, `level ...`, or a --flag)", lineNo, dim)
	}
}

func parseMatchDim(s string, lineNo int) (matchDim, error) {
	switch s {
	case "level":
		return dimLevel, nil
	case "exit":
		return dimExit, nil
	case "":
		return 0, fmt.Errorf("line %d: `match` needs a dimension (level|exit)", lineNo)
	default:
		return 0, fmt.Errorf("line %d: unknown match dimension `%s` (expected level|exit; flags must use `if --flag:`)", lineNo, s)
	}
}

func buildMatchAtom(dim matchDim, value string, lineNo int) (Atom, error) {
	switch dim {
	case dimLevel:
		lvl, err := ParseLevel(value)
		if err != nil {
			return Atom{}, fmt.Errorf("line %d: %w", lineNo, err)
		}
		return Atom{Kind: AtomLevel, Level: lvl}, nil
	default: // dimExit
		switch value {
		case "ok":
			return Atom{Kind: AtomExit, Exit: ExitOk}, nil
		case "failed":
			return Atom{Kind: AtomExit, Exit: ExitFailed}, nil
		default:
			return Atom{}, fmt.Errorf("line %d: unknown exit value `%s` (expected ok|failed)", lineNo, value)
		}
	}
}

func parseDefineHeader(s string) (string, []string, string, error) {
	s = strings.TrimLeft(s, " \t")
	end := strings.IndexFunc(s, func(c rune) bool {
		return c == '(' || c == ':' || unicode.IsSpace(c)
	})
	if end < 0 {
		end = len(s)
	}
	name := s[:end]
	if name == "" {
		return "", nil, "", fmt.Errorf("define needs a name")
	}
	rest := strings.TrimLeft(s[end:], " \t")
	if r, ok := strings.CutPrefix(rest, "("); ok {
		close := strings.IndexByte(r, ')')
		if close < 0 {
			return "", nil, "", fmt.Errorf("missing `)` in define params")
		}
		var params []string
		for _, p := range strings.Split(r[:close], ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				params = append(params, p)
			}
		}
		return name, params, strings.TrimLeft(r[close+1:], " \t"), nil
	}
	return name, nil, rest, nil
}

type regexLit struct {
	src string
	re  *regexp.Regexp
}

func parseRegexLiteral(s string, lineNo int) (regexLit, error) {
	re, after, err := parseRegexLiteralAndRest(s, lineNo)
	if err != nil {
		return regexLit{}, err
	}
	if strings.TrimSpace(after) != "" {
		return regexLit{}, fmt.Errorf("line %d: unexpected trailing input after regex: `%s`", lineNo, strings.TrimSpace(after))
	}
	return re, nil
}

func parseRegexLiteralAndRest(s string, lineNo int) (regexLit, string, error) {
	s = strings.TrimLeft(s, " \t")
	if !strings.HasPrefix(s, "/") {
		return regexLit{}, "", fmt.Errorf("line %d: expected `/regex/`, got `%s`", lineNo, preview(s))
	}
	body := s[1:]
	var src strings.Builder
	end := -1
	for i := 0; i < len(body); i++ {
		c := body[i]
		if c == '\\' {
			if i+1 < len(body) {
				n := body[i+1]
				if n == '/' {
					src.WriteByte('/')
				} else {
					src.WriteByte('\\')
					src.WriteByte(n)
				}
				i++
			} else {
				return regexLit{}, "", fmt.Errorf("line %d: trailing backslash in regex", lineNo)
			}
		} else if c == '/' {
			end = i
			break
		} else {
			src.WriteByte(c)
		}
	}
	if end == -1 {
		return regexLit{}, "", fmt.Errorf("line %d: unterminated regex", lineNo)
	}
	after := body[end+1:]
	re, err := regexp.Compile(src.String())
	if err != nil {
		return regexLit{}, "", fmt.Errorf("line %d: invalid regex `%s`: %w", lineNo, src.String(), err)
	}
	return regexLit{src: src.String(), re: re}, after, nil
}

func parseStringLiteral(s string, lineNo int) (string, error) {
	str, after, err := parseStringLiteralAndRest(s, lineNo)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(after) != "" {
		return "", fmt.Errorf("line %d: unexpected trailing input after string: `%s`", lineNo, strings.TrimSpace(after))
	}
	return str, nil
}

func parseStringLiteralAndRest(s string, lineNo int) (string, string, error) {
	s = strings.TrimLeft(s, " \t")
	if !strings.HasPrefix(s, "\"") {
		return "", "", fmt.Errorf("line %d: expected `\"...\"`, got `%s`", lineNo, preview(s))
	}
	body := s[1:]
	var out strings.Builder
	end := -1
	for i := 0; i < len(body); i++ {
		c := body[i]
		if c == '\\' {
			if i+1 < len(body) {
				switch body[i+1] {
				case 'n':
					out.WriteByte('\n')
				case 't':
					out.WriteByte('\t')
				case 'r':
					out.WriteByte('\r')
				case '\\':
					out.WriteByte('\\')
				case '"':
					out.WriteByte('"')
				default:
					out.WriteByte('\\')
					out.WriteByte(body[i+1])
				}
				i++
			} else {
				return "", "", fmt.Errorf("line %d: trailing backslash in string", lineNo)
			}
		} else if c == '"' {
			end = i
			break
		} else {
			out.WriteByte(c)
		}
	}
	if end == -1 {
		return "", "", fmt.Errorf("line %d: unterminated string", lineNo)
	}
	return out.String(), body[end+1:], nil
}

func parseHeadArg(s string, lineNo int) (auto bool, n int, err error) {
	s = strings.TrimSpace(s)
	if s == "auto" {
		return true, 0, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return false, 0, fmt.Errorf("line %d: expected number or `auto`, got `%s`", lineNo, s)
	}
	return false, v, nil
}

func parseMacroArgs(s string, lineNo int) ([]MacroArg, error) {
	var out []MacroArg
	rest := strings.TrimSpace(s)
	for rest != "" {
		if strings.HasPrefix(rest, "\"") {
			sv, after, err := parseStringLiteralAndRest(rest, lineNo)
			if err != nil {
				return nil, err
			}
			out = append(out, MacroArg{Str: sv})
			rest = strings.TrimLeft(after, " \t")
		} else {
			word, after := takeWord(rest)
			out = append(out, macroArgFromWord(word))
			rest = strings.TrimLeft(after, " \t")
		}
	}
	return out, nil
}

func parseMacroArgsUntilOp(s string, macroNames []string, lineNo int) ([]MacroArg, string, error) {
	var out []MacroArg
	rest := strings.TrimLeft(s, " \t")
	for rest != "" {
		word, _ := takeWord(rest)
		if opKeywords[word] || contains(macroNames, word) {
			break
		}
		if strings.HasPrefix(rest, "\"") {
			sv, after, err := parseStringLiteralAndRest(rest, lineNo)
			if err != nil {
				return nil, "", err
			}
			out = append(out, MacroArg{Str: sv})
			rest = strings.TrimLeft(after, " \t")
		} else {
			w, after := takeWord(rest)
			out = append(out, macroArgFromWord(w))
			rest = strings.TrimLeft(after, " \t")
		}
	}
	return out, rest, nil
}

func macroArgFromWord(w string) MacroArg {
	if n, err := strconv.Atoi(w); err == nil {
		return MacroArg{IsNumber: true, Number: n}
	}
	return MacroArg{Str: w}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func preview(s string) string {
	if len(s) > 40 {
		return s[:40]
	}
	return s
}
