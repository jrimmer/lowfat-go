package ls

import (
	"testing"

	"github.com/jrimmer/lowfat-go/lf"
)

func TestLastField(t *testing.T) {
	s := shell{}
	out, err := s.RunShell("awk '{print $NF}'", "-rw-r--r-- 1 u g 12 Jan 1 file.txt\ntotal 8\n", &lf.ExecCtx{Level: lf.Ultra})
	if err != nil {
		t.Fatal(err)
	}
	if out != "file.txt\n8" {
		t.Errorf("got %q", out)
	}
}

func TestCompactLongForm(t *testing.T) {
	in := "total 16\n" +
		"-rw-r--r--  1 user group  1024 Jan  1 12:00 readme.md\n" +
		"drwxr-xr-x  2 user group  4096 Jan  1 12:00 src\n" +
		"plain-line-not-long-form\n"
	got := compactLongForm(in)
	want := "total 16\n" +
		"- 1024 readme.md\n" +
		"d 4096 src\n" +
		"plain-line-not-long-form"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestCompactLongFormNameWithSpaces(t *testing.T) {
	in := "-rw-r--r--  1 user group  10 Jan  1 12:00 my file.txt\n"
	got := compactLongForm(in)
	if want := "- 10 my file.txt"; got != want {
		t.Errorf("name with spaces: got %q want %q", got, want)
	}
}
