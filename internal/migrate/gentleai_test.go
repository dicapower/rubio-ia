package migrate

import (
	"bytes"
	"encoding/json"
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

// fixture mirrors a real Gentle-AI install: nested blocks, an output style, a
// permissions section and generic skills that must survive.
func fixture(t *testing.T) (home, claudeMD, settings string) {
	home = t.TempDir()
	claudeMD = filepath.Join(home, ".claude", "CLAUDE.md")
	settings = filepath.Join(home, ".claude", "settings.json")
	put(t, claudeMD, "# Mine\n\nKeep this.\n\n"+
		"<!-- gentle-ai:persona -->\nSpeak in voseo.\n<!-- /gentle-ai:persona -->\n\n"+
		"<!-- gentle-ai:sdd-orchestrator -->\nOrchestrate.\n"+
		"<!-- gentle-ai:sdd-model-assignments -->\ntable\n<!-- /gentle-ai:sdd-model-assignments -->\n"+
		"More.\n<!-- /gentle-ai:sdd-orchestrator -->\n\n"+
		"<!-- gentle-ai:strict-tdd-mode -->\nStrict TDD Mode: enabled\n<!-- /gentle-ai:strict-tdd-mode -->\n")
	put(t, settings, `{"outputStyle":"Gentleman","enabledPlugins":{"engram@engram":true,"other@x":true},"permissions":{"defaultMode":"default","deny":["Read(.env)"]}}`)
	put(t, filepath.Join(home, ".cursor", "rules", "gentle-ai.mdc"), "rule")
	put(t, filepath.Join(home, ".claude", "skills", "sdd-init", "SKILL.md"), "sdd")
	put(t, filepath.Join(home, ".claude", "skills", "_shared", "SKILL.md"), "shared")
	put(t, filepath.Join(home, ".claude", "skills", "branch-pr", "SKILL.md"), "generic, keep")
	put(t, filepath.Join(home, ".claude", "agents", "sdd-apply.md"), "agent")
	put(t, filepath.Join(home, ".claude", "commands", "sdd-new.md"), "command")
	put(t, filepath.Join(home, ".claude", "output-styles", "gentleman.md"), "style")
	return
}

func TestPlanOnlyByDefault(t *testing.T) {
	home, claudeMD, _ := fixture(t)
	before := get(t, claudeMD)
	var out bytes.Buffer
	if err := GentleAI(home, Options{}, &out); err != nil {
		t.Fatal(err)
	}
	if get(t, claudeMD) != before || exists(filepath.Join(home, ".rubio-ia")) {
		t.Error("the default run must not change anything")
	}
	for _, want := range []string{"Plan:", "persona", "sdd-orchestrator", "Gentleman output style", "--yes", "--skills"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("plan lacks %q:\n%s", want, out.String())
		}
	}
}

func TestApplyKeepsUserContentAndGenericSkills(t *testing.T) {
	home, claudeMD, settings := fixture(t)
	var out bytes.Buffer
	if err := GentleAI(home, Options{Yes: true}, &out); err != nil {
		t.Fatal(err)
	}
	got := get(t, claudeMD)
	if strings.Contains(got, "gentle-ai") || strings.Contains(got, "voseo") || strings.Contains(got, "Orchestrate") {
		t.Errorf("Gentle-AI text left behind:\n%s", got)
	}
	if got != "# Mine\n\nKeep this.\n" {
		t.Errorf("user text damaged or spacing not tidy: %q", got)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(get(t, settings)), &m); err != nil {
		t.Fatal(err)
	}
	if _, has := m["outputStyle"]; has {
		t.Error("output style not unset")
	}
	if p := m["enabledPlugins"].(map[string]any); p["engram@engram"] != true || p["other@x"] != true {
		t.Error("plugins must be untouched without --plugins")
	}
	if m["permissions"].(map[string]any)["deny"] == nil {
		t.Error("unrelated settings were lost")
	}
	if exists(filepath.Join(home, ".cursor", "rules", "gentle-ai.mdc")) {
		t.Error("cursor rule not deleted")
	}
	if !exists(filepath.Join(home, ".claude", "skills", "sdd-init")) {
		t.Error("SDD skills must stay without --skills")
	}
}

func TestSkillsAndPluginsFlags(t *testing.T) {
	home, _, settings := fixture(t)
	var out bytes.Buffer
	if err := GentleAI(home, Options{Yes: true, Skills: true, Plugins: true}, &out); err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{
		"skills/sdd-init", "skills/_shared", "agents/sdd-apply.md", "commands/sdd-new.md", "output-styles/gentleman.md",
	} {
		if exists(filepath.Join(home, ".claude", gone)) {
			t.Errorf("%s should be gone", gone)
		}
	}
	if !exists(filepath.Join(home, ".claude", "skills", "branch-pr", "SKILL.md")) {
		t.Error("a generic skill was removed")
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(get(t, settings)), &m)
	if m["enabledPlugins"].(map[string]any)["engram@engram"] != false {
		t.Error("engram plugin not disabled")
	}
}

func TestRestoreUndoesTheMigration(t *testing.T) {
	home, claudeMD, settings := fixture(t)
	origMD, origSettings := get(t, claudeMD), get(t, settings)
	var out bytes.Buffer
	if err := GentleAI(home, Options{Yes: true, Skills: true, Plugins: true}, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := backup.Restore(home, ""); err != nil {
		t.Fatal(err)
	}
	if get(t, claudeMD) != origMD || get(t, settings) != origSettings {
		t.Error("files not restored byte for byte")
	}
	for _, back := range []string{
		".claude/skills/sdd-init/SKILL.md", ".claude/agents/sdd-apply.md", ".cursor/rules/gentle-ai.mdc",
	} {
		if !exists(filepath.Join(home, filepath.FromSlash(back))) {
			t.Errorf("%s not restored", back)
		}
	}
}

func TestNothingToDo(t *testing.T) {
	home := t.TempDir()
	var out bytes.Buffer
	if err := GentleAI(home, Options{Yes: true}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Nothing from Gentle-AI") {
		t.Errorf("output: %s", out.String())
	}
}

func TestInvalidSettingsAreLeftAlone(t *testing.T) {
	home := t.TempDir()
	settings := filepath.Join(home, ".claude", "settings.json")
	put(t, settings, "{ not json")
	put(t, filepath.Join(home, ".claude", "CLAUDE.md"), "<!-- gentle-ai:persona -->\nx\n<!-- /gentle-ai:persona -->\n")
	var out bytes.Buffer
	if err := GentleAI(home, Options{Yes: true}, &out); err != nil {
		t.Fatal(err)
	}
	if get(t, settings) != "{ not json" || !strings.Contains(out.String(), "settings left alone") {
		t.Errorf("invalid settings were modified: %s", out.String())
	}
}
