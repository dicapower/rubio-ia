package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func put(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func has(fs []Finding, l Level, sub string) bool {
	for _, f := range fs {
		if f.Level == l && strings.Contains(f.Msg, sub) {
			return true
		}
	}
	return false
}

func TestNoAgent(t *testing.T) {
	if !has(Run(t.TempDir()), Warn, "no supported agent") {
		t.Error("expected a warning when no agent is installed")
	}
}

func TestInstalledCoreIsReported(t *testing.T) {
	home := t.TempDir()
	put(t, filepath.Join(home, ".claude", "CLAUDE.md"), "x\n\n<!-- rubio-ia:core -->\nhello\n<!-- /rubio-ia:core -->\n")
	fs := Run(home)
	if !has(fs, OK, "core installed") || has(fs, Warn, "not installed") {
		t.Errorf("findings: %v", fs)
	}
}

func TestMissingCoreAndConflicts(t *testing.T) {
	home := t.TempDir()
	put(t, filepath.Join(home, ".claude", "CLAUDE.md"), "<!-- gentle-ai:persona -->\nx\n<!-- /gentle-ai:persona -->\n")
	put(t, filepath.Join(home, ".claude", "settings.json"),
		`{"outputStyle":"Gentleman","enabledPlugins":{"engram@engram":true},"permissions":{"defaultMode":"bypassPermissions"}}`)
	put(t, filepath.Join(home, ".cursor", "rules", "gentle-ai.mdc"), "r")
	fs := Run(home)
	for _, want := range []struct {
		l   Level
		sub string
	}{
		{Warn, "not installed"},
		{Warn, "Gentle-AI blocks present"},
		{Warn, "output style"},
		{Info, "Engram plugin"},
		{Warn, "bypassPermissions"},
		{Warn, "Gentle-AI rule file"},
	} {
		if !has(fs, want.l, want.sub) {
			t.Errorf("missing %s finding containing %q in %v", want.l, want.sub, fs)
		}
	}
}

func TestHeavyInstructionsAreFlagged(t *testing.T) {
	home := t.TempDir()
	put(t, filepath.Join(home, ".claude", "CLAUDE.md"), strings.Repeat("a", heavyTokens*4+100))
	if !has(Run(home), Warn, "sent with every request") {
		t.Error("expected a heavy-instructions warning")
	}
}

func TestInvalidSettingsJSON(t *testing.T) {
	home := t.TempDir()
	put(t, filepath.Join(home, ".claude", "settings.json"), "{ nope")
	if !has(Run(home), Warn, "not valid JSON") {
		t.Error("expected an invalid JSON warning")
	}
}
