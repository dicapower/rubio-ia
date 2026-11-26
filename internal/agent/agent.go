// Package agent knows where each supported coding agent keeps its global
// instructions and skills.
package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dicapower/rubio-ia/internal/block"
)

const (
	// Prefix and BlockName make the markers `<!-- rubio-ia:core -->`.
	Prefix    = "rubio-ia"
	BlockName = "core"

	cursorHeader = "---\ndescription: Rubio-IA working agreement\nalwaysApply: true\n---\n\n"
)

// Names are the agents Rubio-IA can configure.
var Names = []string{"claude", "cursor"}

// Agent is one agent installation under a home directory.
type Agent struct {
	Name string
	Home string
}

// New validates an agent name.
func New(name, home string) (Agent, error) {
	for _, n := range Names {
		if n == name {
			return Agent{Name: name, Home: home}, nil
		}
	}
	return Agent{}, fmt.Errorf("unknown agent %q (choose one of: %s, all)", name, strings.Join(Names, ", "))
}

// All returns every supported agent, installed or not.
func All(home string) []Agent {
	out := make([]Agent, 0, len(Names))
	for _, n := range Names {
		out = append(out, Agent{Name: n, Home: home})
	}
	return out
}

// Dir is the agent's configuration folder.
func (a Agent) Dir() string {
	if a.Name == "cursor" {
		return filepath.Join(a.Home, ".cursor")
	}
	return filepath.Join(a.Home, ".claude")
}

// Detected reports whether the agent's folder exists.
func (a Agent) Detected() bool {
	info, err := os.Stat(a.Dir())
	return err == nil && info.IsDir()
}

// Instructions is the file that carries the always-loaded core.
func (a Agent) Instructions() string {
	if a.Name == "cursor" {
		return filepath.Join(a.Dir(), "rules", "rubio-ia.mdc")
	}
	return filepath.Join(a.Dir(), "CLAUDE.md")
}

// SkillsDir is where the agent looks for skills.
func (a Agent) SkillsDir() string { return filepath.Join(a.Dir(), "skills") }

// Compose returns the instructions file with the core block installed.
func (a Agent) Compose(existing, core string) string {
	if a.Name == "cursor" && existing == "" {
		return cursorHeader + block.Wrap(Prefix, BlockName, core)
	}
	return block.Upsert(existing, Prefix, BlockName, core)
}

// Strip returns the instructions file without the core block. An empty result
// means the file held nothing else and can be deleted.
func (a Agent) Strip(existing string) (string, bool) {
	rest, ok := block.Remove(existing, Prefix, BlockName)
	if !ok {
		return existing, false
	}
	empty := strings.TrimSpace(rest) == ""
	if a.Name == "cursor" && strings.TrimSpace(rest) == strings.TrimSpace(cursorHeader) {
		empty = true
	}
	if empty {
		return "", true
	}
	return rest, true
}
