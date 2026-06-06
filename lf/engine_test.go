package lf

import "testing"

// run is a test helper: parse src, execute against input at ctx, return output.
func run(t *testing.T, src, sub string, level Level, exit int, args []string, input string) string {
	t.Helper()
	rs, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := Execute(rs, &ExecCtx{Sub: sub, Level: level, ExitCode: exit, Args: args}, input)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	return out
}

func TestKeepDropHead(t *testing.T) {
	src := "status:\n    keep /^M /\n    head 2\n"
	got := run(t, src, "status", Full, 0, nil, "M one\n?? two\nM three\nM four\nM five\n")
	if want := "M one\nM three\n"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestTail(t *testing.T) {
	src := "logs:\n    tail 2\n"
	got := run(t, src, "logs", Full, 0, nil, "a\nb\nc\nd\n")
	if want := "c\nd\n"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestOrFallbackOnEmpty(t *testing.T) {
	src := "status:\n    keep /NEVER/\n    or \"clean\"\n"
	got := run(t, src, "status", Full, 0, nil, "M one\nM two\n")
	if want := "clean\n"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestNoMatchPassthrough(t *testing.T) {
	src := "specific:\n    head 1\n"
	got := run(t, src, "other", Full, 0, nil, "a\nb\nc\n")
	if want := "a\nb\nc\n"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestMatchLevel(t *testing.T) {
	src := "x:\n    match level:\n        ultra: head 1\n        lite:  head 3\n        else:  head 2\n"
	in := "a\nb\nc\nd\n"
	cases := map[Level]string{Ultra: "a\n", Lite: "a\nb\nc\n", Full: "a\nb\n"}
	for lvl, want := range cases {
		if got := run(t, src, "x", lvl, 0, nil, in); got != want {
			t.Errorf("level %v: got %q want %q", lvl, got, want)
		}
	}
}

func TestIfExitFailedRaw(t *testing.T) {
	src := "x:\n    if exit failed:\n        raw\n    else:\n        head 1\n"
	in := "a\nb\nc\n"
	if got := run(t, src, "x", Full, 1, nil, in); got != in {
		t.Errorf("exit!=0 should be raw: got %q", got)
	}
	if got := run(t, src, "x", Full, 0, nil, in); got != "a\n" {
		t.Errorf("exit==0 should head 1: got %q", got)
	}
}

func TestFlagGuard(t *testing.T) {
	src := "diff:\n    if --stat:\n        head 1\n    else:\n        head 2\n"
	in := "a\nb\nc\n"
	if got := run(t, src, "diff", Full, 0, []string{"diff", "--stat"}, in); got != "a\n" {
		t.Errorf("--stat present: got %q", got)
	}
	if got := run(t, src, "diff", Full, 0, []string{"diff"}, in); got != "a\nb\n" {
		t.Errorf("--stat absent: got %q", got)
	}
}

func TestFlagGuardValueAndGlued(t *testing.T) {
	if !flagMatches("-o yaml", []string{"get", "-o", "yaml"}) {
		t.Error("two-token -o yaml should match")
	}
	if !flagMatches("-o yaml", []string{"get", "-oyaml"}) {
		t.Error("glued -oyaml should match")
	}
	if !flagMatches("--output json", []string{"--output=json"}) {
		t.Error("--output=json should match")
	}
	if flagMatches("--stat", []string{"--statistics"}) {
		t.Error("--stat must not match --statistics")
	}
}

func TestSplitPrePost(t *testing.T) {
	// pre: keep commit lines; post: keep + lines. join with newline.
	src := "show:\n    split /^diff /\n    pre:\n        keep /^commit/\n    post:\n        keep /^[+]/\n"
	in := "commit abc\nAuthor: x\ndiff --git a b\n+added\n-removed\n context\n"
	got := run(t, src, "show", Full, 0, nil, in)
	if want := "commit abc\n+added\n"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestMacroParameterless(t *testing.T) {
	src := "define keepM:\n    keep /^M/\n\nx:\n    keepM\n    head 2\n"
	got := run(t, src, "x", Full, 0, nil, "M one\n?? two\nM three\nM four\n")
	if want := "M one\nM three\n"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// captureShell records the (expanded) command it is asked to run.
type captureShell struct{ lastCmd string }

func (c *captureShell) RunShell(cmd, stdin string, ctx *ExecCtx) (string, error) {
	c.lastCmd = cmd
	return stdin, nil
}

func TestMacroParamExpandsInShellBody(t *testing.T) {
	// $1 is substituted into shell: bodies (and only there), matching Rust's
	// expand_args which runs for shell/python/or-shell ops exclusively.
	src := "define run(n):\n    shell: awk -v lim=$1 'x'\n\ndiff:\n    run 200\n"
	rs, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	cap := &captureShell{}
	if _, err := Execute(rs, &ExecCtx{Sub: "diff", Level: Full, Shell: cap}, "in\n"); err != nil {
		t.Fatal(err)
	}
	if cap.lastCmd != "awk -v lim=200 'x'" {
		t.Errorf("expected $1 expanded to 200, got %q", cap.lastCmd)
	}
}

func TestHeadAutoLevelScaling(t *testing.T) {
	// head auto => level.HeadLimit(30): ultra=15, full=30, lite=60.
	src := "x:\n    head auto\n"
	in := ""
	for i := 0; i < 100; i++ {
		in += "line\n"
	}
	count := func(s string) int {
		n := 0
		for _, c := range s {
			if c == '\n' {
				n++
			}
		}
		return n
	}
	if got := count(run(t, src, "x", Ultra, 0, nil, in)); got != 15 {
		t.Errorf("ultra head auto: got %d want 15", got)
	}
	if got := count(run(t, src, "x", Full, 0, nil, in)); got != 30 {
		t.Errorf("full head auto: got %d want 30", got)
	}
	if got := count(run(t, src, "x", Lite, 0, nil, in)); got != 60 {
		t.Errorf("lite head auto: got %d want 60", got)
	}
}

func TestGlobSubMatch(t *testing.T) {
	src := "get-*:\n    head 1\n"
	if got := run(t, src, "get-pods", Full, 0, nil, "a\nb\n"); got != "a\n" {
		t.Errorf("glob get-* should match get-pods: got %q", got)
	}
	if got := run(t, src, "delete", Full, 0, nil, "a\nb\n"); got != "a\nb\n" {
		t.Errorf("glob get-* should not match delete: got %q", got)
	}
}

func TestShellOpWithoutRunnerErrors(t *testing.T) {
	rs, err := Parse("x:\n    shell: sed s/a/b/\n")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Execute(rs, &ExecCtx{Sub: "x", Level: Full}, "a\n")
	if err == nil {
		t.Error("expected error when shell op runs with nil ShellRunner")
	}
}

func TestParseErrors(t *testing.T) {
	bad := []string{
		"x\n    head 1\n",     // missing colon in selector
		"x:\n    keep foo\n",  // regex not /.../
		"x:\n    bogusop 1\n", // unknown op
		"x:\n",                // rule with no ops
	}
	for _, src := range bad {
		if _, err := Parse(src); err == nil {
			t.Errorf("expected parse error for %q", src)
		}
	}
}

func TestEnsureTrailingNewline(t *testing.T) {
	// non-empty output gets exactly one trailing newline
	got := run(t, "x:\n    head 1\n", "x", Full, 0, nil, "only")
	if got != "only\n" {
		t.Errorf("got %q want %q", got, "only\n")
	}
	// empty output stays empty (no spurious newline)
	got = run(t, "x:\n    keep /Z/\n", "x", Full, 0, nil, "abc\n")
	if got != "" {
		t.Errorf("empty output should stay empty, got %q", got)
	}
}
