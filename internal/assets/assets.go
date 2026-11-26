// Package assets holds the instructions Rubio-IA installs. They are embedded
// so the binary needs no network and no files next to it.
package assets

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed core.md skills/plan/SKILL.md
var files embed.FS

// SkillMarker is written inside every installed skill. Remove and migrate use
// it to tell a skill they installed from one the user wrote.
const SkillMarker = "<!-- rubio-ia:skill -->"

var langRules = map[string]string{
	"auto":  "Reply in the language the user writes in, keeping their regional variety.",
	"es-ES": "Reply in Spanish from Spain (tuteo and vosotros; never voseo or Latin American regionalisms) unless the user asks for another language.",
	"en":    "Reply in English unless the user writes in another language.",
}

// Languages returns the values --lang accepts, sorted.
func Languages() []string {
	out := make([]string, 0, len(langRules))
	for k := range langRules {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Core returns the always-loaded instructions for a language rule.
func Core(lang string) (string, error) {
	if lang == "" {
		lang = "auto"
	}
	rule, ok := langRules[lang]
	if !ok {
		return "", fmt.Errorf("unknown language %q (choose one of: %s)", lang, strings.Join(Languages(), ", "))
	}
	raw, err := files.ReadFile("core.md")
	if err != nil {
		return "", fmt.Errorf("read core: %w", err)
	}
	return strings.TrimSpace(strings.ReplaceAll(string(raw), "{{LANG}}", rule)) + "\n", nil
}

// Skills returns the optional skills, keyed by the directory name they are
// installed under.
func Skills() (map[string]string, error) {
	raw, err := files.ReadFile("skills/plan/SKILL.md")
	if err != nil {
		return nil, fmt.Errorf("read plan skill: %w", err)
	}
	return map[string]string{"rubio-plan": string(raw)}, nil
}
