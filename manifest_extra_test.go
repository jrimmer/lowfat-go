package lowfat

import (
	"strings"
	"testing"
)

func TestNewFilterValidationAndParseErrors(t *testing.T) {
	if _, err := NewFilter(Manifest{Commands: []string{"x"}}, "", nil); err == nil || !strings.Contains(err.Error(), "missing Name") {
		t.Fatalf("missing name error = %v", err)
	}
	if _, err := NewFilter(Manifest{Name: "x"}, "", nil); err == nil || !strings.Contains(err.Error(), "declares no Commands") {
		t.Fatalf("missing commands error = %v", err)
	}
	if _, err := NewFilter(Manifest{Name: "x", Commands: []string{"x"}}, "match [", nil); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Fatalf("parse error = %v", err)
	}
}

func TestMustNewFilterPanicsOnInvalidManifest(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("MustNewFilter did not panic")
		}
	}()
	_ = MustNewFilter(Manifest{}, "", nil)
}

func TestRegistryCommandsAndDuplicateErrors(t *testing.T) {
	f := MustNewFilter(Manifest{Name: "x", Commands: []string{"x"}}, "", nil)
	r := NewRegistry()
	if err := r.Register(nil); err == nil || !strings.Contains(err.Error(), "nil filter") {
		t.Fatalf("nil register error = %v", err)
	}
	if err := r.Register(&ToolFilter{manifest: Manifest{Name: "empty"}}); err == nil || !strings.Contains(err.Error(), "declares no commands") {
		t.Fatalf("empty commands error = %v", err)
	}
	if err := r.Register(f); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := r.Register(f); err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("duplicate error = %v", err)
	}
	cmds := r.Commands()
	if len(cmds) != 1 || cmds[0] != "x" {
		t.Fatalf("Commands() = %#v", cmds)
	}
	if got := f.Manifest(); got.Name != "x" || len(got.Commands) != 1 || got.Commands[0] != "x" {
		t.Fatalf("Manifest() = %#v", got)
	}
}
