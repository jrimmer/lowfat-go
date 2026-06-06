package rg

import "testing"

func TestNewManifest(t *testing.T) {
	f := New()
	if f == nil {
		t.Fatal("New returned nil")
	}
	m := f.Manifest()
	if m.Name == "" {
		t.Fatal("manifest name is empty")
	}
	if len(m.Commands) == 0 {
		t.Fatal("manifest commands are empty")
	}
}
