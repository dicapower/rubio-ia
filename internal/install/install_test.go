package install

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dicapower/rubio-ia/internal/backup"
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

func get(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func exists(path string) bool { _, err := os.Stat(path); return err == nil }

func TestSetupKeepsUserTextAndIsIdempotent(t *testing.T) {
	home := t.TempDir()
	claude := filepath.Join(home, ".claude", "CLAUDE.md")
	put(t, claude, "# My rules\n\nBe kind.\n")

	var out bytes.Buffer
	o := Options{Lang: "es-ES", Skills: true}
	if err := Setup(home, o, &out); err != nil {
		t.Fatal(err)
	}
	got := get(t, claude)
	if !strings.Contains(got, "Be kind.") || !strings.Contains(got, "<!-- rubio-ia:core -->") {
		t.Fatalf("unexpected file:\n%s", got)
	}
	if !strings.Contains(got, "Spain") {
		t.Error("the language rule was not applied")
	}
	if !exists(filepath.Join(home, ".claude", "skills", "rubio-plan", "SKILL.md")) {
		t.Error("skill not installed")
	}
	if exists(filepath.Join(home, ".cursor")) {
		t.Error("an agent that is not installed must be left alone")
	}

	out.Reset()
	if err := Setup(home, o, &out); err != nil {
		t.Fatal(err)
	}
	if get(t, claude) != got || !strings.Contains(out.String(), "already up to date") {
		t.Errorf("second setup changed things:\n%s", out.String())
	}
	if all, _ := backup.List(home); len(all) != 1 {
		t.Errorf("a no-op setup must not create a backup, have %d", len(all))
	}
}

func TestRemoveRestoresOriginalAndRestoreUndoes(t *testing.T) {
	home := t.TempDir()
	claude := filepath.Join(home, ".claude", "CLAUDE.md")
	orig := "# My rules\n\nBe kind.\n"
	put(t, claude, orig)
	var out bytes.Buffer
	if err := Setup(home, Options{Skills: true}, &out); err != nil {
		t.Fatal(err)
	}
	installed := get(t, claude)

	if err := Remove(home, nil, false, &out); err != nil {
		t.Fatal(err)
	}
	if get(t, claude) != orig {
		t.Errorf("remove did not restore the original:\n%q", get(t, claude))
	}
	if exists(filepath.Join(home, ".claude", "skills", "rubio-plan")) {
		t.Error("skill not removed")
	}

	if _, err := backup.Restore(home, ""); err != nil { // undo the remove
		t.Fatal(err)
	}
	if get(t, claude) != installed {
		t.Error("restore did not bring back the installed state")
	}
}

func TestCursorFileIsCreatedAndDeleted(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Setup(home, Options{Agents: []string{"cursor"}}, &out); err != nil {
		t.Fatal(err)
	}
	rule := filepath.Join(home, ".cursor", "rules", "rubio-ia.mdc")
	if got := get(t, rule); !strings.HasPrefix(got, "---\n") || !strings.Contains(got, "alwaysApply: true") {
		t.Errorf("missing cursor frontmatter:\n%s", got)
	}
	if err := Remove(home, []string{"cursor"}, false, &out); err != nil {
		t.Fatal(err)
	}
	if exists(rule) {
		t.Error("a rule file that only held our block must be deleted")
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Setup(home, Options{Skills: true, DryRun: true}, &out); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(home, ".claude", "CLAUDE.md")) || exists(filepath.Join(home, ".rubio-ia")) {
		t.Error("dry run touched the disk")
	}
	if !strings.Contains(out.String(), "would write") {
		t.Errorf("output: %s", out.String())
	}
}

func TestUserSkillWithSameNameIsNotOverwritten(t *testing.T) {
	home := t.TempDir()
	skill := filepath.Join(home, ".claude", "skills", "rubio-plan", "SKILL.md")
	put(t, skill, "my own skill")
	var out bytes.Buffer
	if err := Setup(home, Options{Skills: true}, &out); err != nil {
		t.Fatal(err)
	}
	if get(t, skill) != "my own skill" || !strings.Contains(out.String(), "not from Rubio-IA") {
		t.Errorf("a user skill was touched: %s", out.String())
	}
	if err := Remove(home, nil, false, &out); err != nil {
		t.Fatal(err)
	}
	if get(t, skill) != "my own skill" {
		t.Error("remove deleted a user skill")
	}
}

func TestErrors(t *testing.T) {
	home := t.TempDir()
	var out bytes.Buffer
	if err := Setup(home, Options{}, &out); err == nil {
		t.Error("expected an error when no agent is installed")
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Setup(home, Options{Agents: []string{"cursor"}}, &out); err == nil {
		t.Error("expected an error for an agent that is not installed")
	}
	if err := Setup(home, Options{Agents: []string{"vim"}}, &out); err == nil {
		t.Error("expected an error for an unknown agent")
	}
	if err := Setup(home, Options{Lang: "klingon"}, &out); err == nil {
		t.Error("expected an error for an unknown language")
	}
}
