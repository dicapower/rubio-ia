package assets

import (
	"regexp"
	"strings"
	"testing"

	"github.com/dicapower/rubio-ia/internal/tokens"
)

// The core is loaded in every session of every project, so its size is part of
// the contract. Raise the cap only with a reason written in the changelog.
const (
	maxCoreTokens  = 1000
	maxSkillTokens = 1500
)

func TestCoreStaysSmall(t *testing.T) {
	for _, lang := range Languages() {
		core, err := Core(lang)
		if err != nil {
			t.Fatal(err)
		}
		if n := tokens.Estimate(core); n > maxCoreTokens {
			t.Errorf("core for %q is ~%d tokens, cap is %d", lang, n, maxCoreTokens)
		}
	}
}

func TestSkillsStaySmallAndMarked(t *testing.T) {
	skills, err := Skills()
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range skills {
		if n := tokens.Estimate(content); n > maxSkillTokens {
			t.Errorf("skill %s is ~%d tokens, cap is %d", name, n, maxSkillTokens)
		}
		if !strings.Contains(content, SkillMarker) {
			t.Errorf("skill %s lacks the ownership marker", name)
		}
		if !strings.HasPrefix(content, "---\nname: "+name+"\n") {
			t.Errorf("skill %s must start with frontmatter naming it", name)
		}
	}
}

func TestUnknownLanguageIsRejected(t *testing.T) {
	if _, err := Core("klingon"); err == nil {
		t.Fatal("expected an error for an unknown language")
	}
}

func TestNoPlaceholderLeft(t *testing.T) {
	for _, lang := range Languages() {
		core, _ := Core(lang)
		if strings.Contains(core, "{{") {
			t.Errorf("core for %q still has a placeholder", lang)
		}
	}
}

// Only tools that noah really has may be named: a protocol that points at a
// missing tool wastes turns.
func TestMemoryToolsExistInNoah(t *testing.T) {
	known := map[string]bool{
		"mem_context": true, "mem_search": true, "mem_get": true,
		"mem_save": true, "mem_forget": true, "mem_spaces": true,
	}
	re := regexp.MustCompile(`mem_[a-z_]+`)
	skills, _ := Skills()
	core, _ := Core("auto")
	texts := []string{core}
	for _, s := range skills {
		texts = append(texts, s)
	}
	for _, text := range texts {
		for _, tool := range re.FindAllString(text, -1) {
			if !known[tool] {
				t.Errorf("%s is not a noah tool", tool)
			}
		}
	}
}

// The Spanish default is Spain Spanish: no voseo in anything we ship.
func TestNoVoseoInShippedText(t *testing.T) {
	re := regexp.MustCompile(`(?i)\b(vos|tenés|podés|querés|sos|acá|che|laburo|dale)\b`)
	skills, _ := Skills()
	core, _ := Core("es-ES")
	texts := map[string]string{"core": core}
	for k, v := range skills {
		texts[k] = v
	}
	for name, text := range texts {
		if m := re.FindString(text); m != "" {
			t.Errorf("%s contains %q", name, m)
		}
	}
}
