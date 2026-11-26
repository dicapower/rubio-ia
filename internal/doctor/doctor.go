// Package doctor reports the state of the installation. It only reads.
package doctor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dicapower/rubio-ia/internal/agent"
	"github.com/dicapower/rubio-ia/internal/block"
	"github.com/dicapower/rubio-ia/internal/tokens"
)

// Level classifies a finding.
type Level string

const (
	OK   Level = "ok"
	Warn Level = "warn"
	Info Level = "info"
)

// Finding is one line of the report.
type Finding struct {
	Level Level
	Msg   string
}

const (
	// heavyTokens is where global instructions start to cost real money: they
	// are sent with every request of every session.
	heavyTokens  = 6000
	bigBackupDir = 500 << 20 // 500 MB
)

// Run inspects the home directory.
func Run(home string) []Finding {
	var out []Finding
	add := func(l Level, format string, a ...any) {
		out = append(out, Finding{l, fmt.Sprintf(format, a...)})
	}

	detected := 0
	for _, a := range agent.All(home) {
		if !a.Detected() {
			continue
		}
		detected++
		path := a.Instructions()
		raw, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			add(Warn, "%s: cannot read %s: %v", a.Name, path, err)
			continue
		}
		text := string(raw)
		if s, e, ok := block.Find(text, agent.Prefix, agent.BlockName); ok {
			add(OK, "%s: Rubio-IA core installed in %s (~%d tokens)", a.Name, path, tokens.Estimate(text[s:e]))
		} else {
			add(Warn, "%s: Rubio-IA core is not installed (run: rubio setup)", a.Name)
		}
		if n := tokens.Estimate(text); n > heavyTokens {
			add(Warn, "%s: %s is ~%d tokens, sent with every request", a.Name, path, n)
		}
		if names := block.Names(text, "gentle-ai"); len(names) > 0 {
			add(Warn, "%s: Gentle-AI blocks present (%s); run: rubio migrate gentle-ai", a.Name, strings.Join(names, ", "))
		}
	}
	if detected == 0 {
		add(Warn, "no supported agent found (looked for ~/.claude and ~/.cursor)")
	}

	claudeSettings(home, add)

	if _, err := os.Stat(filepath.Join(home, ".cursor", "rules", "gentle-ai.mdc")); err == nil {
		add(Warn, "cursor: the Gentle-AI rule file is still present (run: rubio migrate gentle-ai)")
	}
	if _, err := exec.LookPath("noah"); err != nil {
		add(Info, "noah is not on PATH: the memory section of the core will be ignored until it is installed")
	} else {
		add(OK, "noah found on PATH")
	}
	if size := dirSize(filepath.Join(home, ".gentle-ai", "backups")); size > bigBackupDir {
		add(Info, "~/.gentle-ai/backups uses %s and may contain chat history; delete it by hand if you no longer need it", human(size))
	}
	return out
}

func claudeSettings(home string, add func(Level, string, ...any)) {
	path := filepath.Join(home, ".claude", "settings.json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		add(Warn, "claude: cannot read %s: %v", path, err)
		return
	}
	var s struct {
		OutputStyle string          `json:"outputStyle"`
		Plugins     map[string]bool `json:"enabledPlugins"`
		Permissions struct {
			DefaultMode string `json:"defaultMode"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		add(Warn, "claude: %s is not valid JSON: %v", path, err)
		return
	}
	if strings.EqualFold(s.OutputStyle, "gentleman") {
		add(Warn, "claude: output style %q is set and forces its own tone (run: rubio migrate gentle-ai)", s.OutputStyle)
	} else if s.OutputStyle != "" {
		add(Info, "claude: output style %q is set", s.OutputStyle)
	}
	if s.Plugins["engram@engram"] {
		add(Info, "claude: the Engram plugin is enabled; its memory instructions can clash with noah's")
	}
	if s.Permissions.DefaultMode == "bypassPermissions" {
		add(Warn, "claude: permission prompts are off (bypassPermissions); keep a deny list for destructive commands")
	}
}

// Print writes the report.
func Print(w io.Writer, fs []Finding) {
	for _, f := range fs {
		fmt.Fprintf(w, "[%s] %s\n", f.Level, f.Msg)
	}
}

func dirSize(root string) int64 {
	var n int64
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			n += info.Size()
		}
		return nil
	})
	return n
}

func human(n int64) string {
	const mb = 1 << 20
	if n >= 1<<30 {
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	}
	return fmt.Sprintf("%d MB", n/mb)
}
