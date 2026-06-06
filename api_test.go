package lowfat_test

import (
	"strings"
	"testing"

	"go.rimmer.net/lowfat"
	_ "go.rimmer.net/lowfat/filters/all"
	"go.rimmer.net/lowfat/filters/git"
)

func TestUnknownCommandPassthrough(t *testing.T) {
	out := "some output\nthat is long enough\n"
	res, err := lowfat.Filter("totally-unknown-cmd", []string{"sub"}, out, lowfat.FullLevel())
	if err != nil {
		t.Fatal(err)
	}
	if res.Filtered {
		t.Error("unknown command should not be filtered")
	}
	if res.Output != out {
		t.Error("unknown command output should pass through unchanged")
	}
	if res.FilterName != "" {
		t.Errorf("unknown command should have no FilterName, got %q", res.FilterName)
	}
}

func TestFilteredReducesTokens(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 200; i++ {
		b.WriteString("commit 0123456789abcdef0123456789abcdef01234567\n")
		b.WriteString("    some commit message body line\n")
		b.WriteString("Signed-off-by: someone <a@b.c>\n")
	}
	res, err := lowfat.Filter("git", []string{"log"}, b.String(), lowfat.FullLevel().At(lowfat.Ultra))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Filtered {
		t.Fatal("git log should be filtered")
	}
	if res.FilterName != "git-compact" {
		t.Errorf("FilterName: got %q want git-compact", res.FilterName)
	}
	if res.OutputTokens >= res.InputTokens {
		t.Errorf("expected token savings: in=%d out=%d", res.InputTokens, res.OutputTokens)
	}
}

func TestMinTokensSkip(t *testing.T) {
	small := "M one\nM two\n" // well under 128 tokens
	res, err := lowfat.Filter("git", []string{"status"}, small,
		lowfat.FullLevel().At(lowfat.Ultra)) // MinTokens default 0 -> filters
	if err != nil {
		t.Fatal(err)
	}
	if !res.Filtered {
		t.Error("with MinTokens=0, small input should still be filtered")
	}

	opt := lowfat.Options{MinTokens: 128}.At(lowfat.Ultra)
	res, err = lowfat.Filter("git", []string{"status"}, small, opt)
	if err != nil {
		t.Fatal(err)
	}
	if res.Filtered {
		t.Error("with MinTokens=128, tiny input should be skipped")
	}
	if res.Output != small {
		t.Error("skipped input should pass through unchanged")
	}
}

func TestDefaultLevelIsFull(t *testing.T) {
	// Build many lines; full keeps more than ultra. Verify zero-value Options
	// (no level set) behaves as Full, not Lite.
	var b strings.Builder
	for i := 0; i < 100; i++ {
		b.WriteString("M file\n")
	}
	full, _ := lowfat.Filter("git", []string{"status"}, b.String(), lowfat.Options{})
	explicitFull, _ := lowfat.Filter("git", []string{"status"}, b.String(), lowfat.FullLevel())
	if full.Output != explicitFull.Output {
		t.Error("zero-value Options should default to Full level")
	}
}

func TestCustomRegistryIsolation(t *testing.T) {
	reg := lowfat.NewRegistry()
	if err := reg.Register(git.New()); err != nil {
		t.Fatal(err)
	}
	// git is registered; docker is not.
	if _, ok := reg.ForCommand("git"); !ok {
		t.Error("git should be registered")
	}
	if _, ok := reg.ForCommand("docker"); ok {
		t.Error("docker should NOT be in this custom registry")
	}
	res, _ := reg.Filter("docker", []string{"ps"}, "long\noutput\nhere\n", lowfat.FullLevel())
	if res.Filtered {
		t.Error("docker should pass through in a registry without it")
	}
}

func TestDuplicateRegistrationFails(t *testing.T) {
	reg := lowfat.NewRegistry()
	if err := reg.Register(git.New()); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(git.New()); err == nil {
		t.Error("registering the same command twice should error")
	}
}

func TestAllExpectedCommandsRegistered(t *testing.T) {
	want := []string{"git", "docker", "ls", "tree", "grep", "find"}
	for _, cmd := range want {
		if _, ok := lowfat.Default().ForCommand(cmd); !ok {
			t.Errorf("expected command %q to be registered by filters/all", cmd)
		}
	}
}
