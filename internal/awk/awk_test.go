package awk

import "testing"

func TestLinesMatchesRustLinesSemantics(t *testing.T) {
	got := Lines("a\r\nb\n")
	want := []string{"a", "b"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Lines() = %#v, want %#v", got, want)
	}
	if got := Lines(""); got != nil {
		t.Fatalf("Lines(empty) = %#v, want nil", got)
	}
}

func TestFieldsAndField(t *testing.T) {
	line := "  alpha\tbeta  gamma "
	if got := Fields(line); len(got) != 3 || got[0] != "alpha" || got[2] != "gamma" {
		t.Fatalf("Fields() = %#v", got)
	}
	cases := []struct {
		idx  int
		want string
	}{
		{0, line},
		{1, "alpha"},
		{3, "gamma"},
		{-1, ""},
		{4, ""},
	}
	for _, tc := range cases {
		if got := Field(line, tc.idx); got != tc.want {
			t.Fatalf("Field(%d) = %q, want %q", tc.idx, got, tc.want)
		}
	}
}

func TestLineTransforms(t *testing.T) {
	input := "a   b\n c    d \n"
	if got, want := CollapseSpaces(input), "a b\n c d "; got != want {
		t.Fatalf("CollapseSpaces() = %q, want %q", got, want)
	}
	if got, want := MapLines("a\nb", func(s string) string { return s + "!" }), "a!\nb!"; got != want {
		t.Fatalf("MapLines() = %q, want %q", got, want)
	}
	if got, want := KeepLines("alpha\nbeta\ngamma", func(s string) bool { return len(s) == 5 }), "alpha\ngamma"; got != want {
		t.Fatalf("KeepLines() = %q, want %q", got, want)
	}
	if got, want := Head("a\nb\nc", 2), "a\nb"; got != want {
		t.Fatalf("Head() = %q, want %q", got, want)
	}
	if got, want := Head("a\nb", 5), "a\nb"; got != want {
		t.Fatalf("Head(long) = %q, want %q", got, want)
	}
}
