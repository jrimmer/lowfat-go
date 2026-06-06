package lf

import (
	"strings"
	"testing"
)

func TestLevelStringAndParse(t *testing.T) {
	cases := []struct {
		in   string
		want Level
		text string
	}{
		{"lite", Lite, "lite"},
		{"FULL", Full, "full"},
		{"ultra", Ultra, "ultra"},
	}
	for _, tc := range cases {
		got, err := ParseLevel(tc.in)
		if err != nil {
			t.Fatalf("ParseLevel(%q): %v", tc.in, err)
		}
		if got != tc.want || got.String() != tc.text {
			t.Fatalf("ParseLevel(%q) = %v/%q, want %v/%q", tc.in, got, got.String(), tc.want, tc.text)
		}
	}
	if Full.String() != "full" || Level(99).String() != "full" {
		t.Fatalf("default String() should be full")
	}
}

func TestParseLevelError(t *testing.T) {
	_, err := ParseLevel("tiny")
	if err == nil || !strings.Contains(err.Error(), "unknown level: tiny") {
		t.Fatalf("ParseLevel invalid error = %v", err)
	}
	if (&ParseError{Msg: "boom"}).Error() != "boom" {
		t.Fatal("ParseError.Error did not return message")
	}
}

func TestHeadLimitBoundaries(t *testing.T) {
	if got := Lite.HeadLimit(10); got != 20 {
		t.Fatalf("lite HeadLimit = %d", got)
	}
	if got := Full.HeadLimit(10); got != 10 {
		t.Fatalf("full HeadLimit = %d", got)
	}
	if got := Ultra.HeadLimit(100); got != 50 {
		t.Fatalf("ultra HeadLimit large = %d", got)
	}
	if got := Ultra.HeadLimit(6); got != 5 {
		t.Fatalf("ultra HeadLimit minimum = %d", got)
	}
}
