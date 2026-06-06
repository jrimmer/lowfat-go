package git

import (
	"testing"

	"go.rimmer.net/lowfat/lf"
)

func TestAbbrevHash(t *testing.T) {
	in := "commit 0123456789abcdef0123456789abcdef01234567\nAuthor: x\n"
	got := abbrevHash(in)
	want := "commit 0123456789ab\nAuthor: x"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestAbbrevHashShortHashUntouched(t *testing.T) {
	in := "commit abc123\n" // not 40 chars -> unchanged
	if got := abbrevHash(in); got != "commit abc123" {
		t.Errorf("short hash should be untouched, got %q", got)
	}
}

func TestCompactDiffFullDropsContext(t *testing.T) {
	in := "diff --git a/f b/f\n@@ -1,3 +1,3 @@ func\n context\n+added\n-removed\n"
	got := compactDiff(in, 100, lf.Full)
	want := "diff --git a/f b/f\n@@ -1,3 +1,3 @@ func\n+added\n-removed"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestCompactDiffUltraTruncatesHunkHeader(t *testing.T) {
	in := "@@ -1,3 +1,3 @@ func foo() {\n+added\n"
	got := compactDiff(in, 100, lf.Ultra)
	// ultra keeps up to and including " @@", dropping the function context;
	// at ultra the +added context line is dropped too.
	want := "@@ -1,3 +1,3 @@"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestCompactDiffLimit(t *testing.T) {
	in := "diff 1\ndiff 2\ndiff 3\ndiff 4\n"
	got := compactDiff(in, 2, lf.Full)
	if want := "diff 1\ndiff 2"; got != want {
		t.Errorf("lim=2: got %q want %q", got, want)
	}
}

func TestShellRunnerDispatch(t *testing.T) {
	s := shell{}
	out, err := s.RunShell("awk 'NF' | head -50", "a\n\nb\n", &lf.ExecCtx{Level: lf.Full})
	if err != nil {
		t.Fatal(err)
	}
	if out != "a\nb" {
		t.Errorf("awk 'NF' should drop blanks: got %q", out)
	}
	if _, err := s.RunShell("rm -rf /", "x", &lf.ExecCtx{}); err == nil {
		t.Error("unrecognized command should error, not silently run")
	}
}
