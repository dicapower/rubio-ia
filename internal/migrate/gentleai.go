// Package migrate removes the instructions another tool left behind, so they
// stop competing with Rubio-IA. It changes nothing unless told to, and it
// backs up every path first.
package migrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dicapower/rubio-ia/internal/backup"
	"github.com/dicapower/rubio-ia/internal/block"
)

// Options for GentleAI.
type Options struct {
	Yes     bool // apply; without it the command only prints the plan
	Skills  bool // also remove the SDD skills, agents, commands and the output style file
	Plugins bool // also disable the Engram plugin in Claude Code settings
}

type step struct {
	desc  string
	paths []string
	run   func() error
}

var blankRuns = regexp.MustCompile(`\n{3,}`)

// GentleAI plans, and with Options.Yes applies, the removal of Gentle-AI's
// instructions from Claude Code and Cursor.
func GentleAI(home string, o Options, out io.Writer) error {
	var steps []step
	var hints []string

	claudeMD := filepath.Join(home, ".claude", "CLAUDE.md")
	if st, err := claudeBlocks(claudeMD); err != nil {
		return err
	} else if st != nil {
		steps = append(steps, *st)
	}

	settings := filepath.Join(home, ".claude", "settings.json")
	if st, err := settingsStep(settings, o.Plugins); err != nil {
		fmt.Fprintf(out, "warning: %v (settings left alone)\n", err)
	} else if st != nil {
		steps = append(steps, *st)
	}

	rule := filepath.Join(home, ".cursor", "rules", "gentle-ai.mdc")
	if _, err := os.Stat(rule); err == nil {
		steps = append(steps, step{
			desc:  "cursor: delete " + rule,
			paths: []string{rule},
			run:   func() error { return os.Remove(rule) },
		})
	}

	sdd := sddPaths(home)
	switch {
	case len(sdd) > 0 && o.Skills:
		steps = append(steps, step{
			desc:  fmt.Sprintf("remove %d SDD skills, agents, commands and the Gentleman output style file", len(sdd)),
			paths: sdd,
			run: func() error {
				for _, p := range sdd {
					if err := os.RemoveAll(p); err != nil {
						return err
					}
				}
				return nil
			},
		})
	case len(sdd) > 0:
		hints = append(hints, fmt.Sprintf("%d SDD skills, agents and commands found; add --skills to remove them", len(sdd)))
	}

	if len(steps) == 0 {
		fmt.Fprintln(out, "Nothing from Gentle-AI to remove in ~/.claude or ~/.cursor.")
		printHints(out, hints)
		return nil
	}

	fmt.Fprintln(out, "Plan:")
	for _, s := range steps {
		fmt.Fprintf(out, "  - %s\n", s.desc)
	}
	printHints(out, hints)
	if !o.Yes {
		fmt.Fprintln(out, "\nNothing changed. Run again with --yes to apply (a backup is taken first).")
		return nil
	}

	sess := backup.Begin(home)
	for _, s := range steps {
		for _, p := range s.paths {
			if err := sess.Add(p); err != nil {
				return err
			}
		}
	}
	for _, s := range steps {
		if err := s.run(); err != nil {
			_ = sess.Commit() // keep the backup so the half-done change can be undone
			return fmt.Errorf("%s: %w (undo with: rubio restore %s)", s.desc, err, sess.ID())
		}
	}
	if err := sess.Commit(); err != nil {
		return err
	}
	fmt.Fprintf(out, "\nDone. Backup %s (undo with: rubio restore %s)\n", sess.ID(), sess.ID())
	fmt.Fprintln(out, "Left alone: ~/.gentle-ai (its state and backups), other skills, and the memory plugin unless --plugins was given.")
	fmt.Fprintln(out, "Next: rubio setup, then restart your agent.")
	return nil
}

func printHints(out io.Writer, hints []string) {
	for _, h := range hints {
		fmt.Fprintf(out, "  hint: %s\n", h)
	}
}

func claudeBlocks(path string) (*step, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	text := string(raw)
	var removable []string
	for _, n := range block.Names(text, "gentle-ai") {
		if _, _, ok := block.Find(text, "gentle-ai", n); ok {
			removable = append(removable, n)
		}
	}
	if len(removable) == 0 {
		return nil, nil
	}
	return &step{
		desc:  fmt.Sprintf("claude: remove Gentle-AI blocks from %s (%s)", path, strings.Join(removable, ", ")),
		paths: []string{path},
		run: func() error {
			rest := text
			for _, n := range removable {
				rest, _ = block.Remove(rest, "gentle-ai", n) // a nested block may already be gone
			}
			rest = strings.Trim(blankRuns.ReplaceAllString(rest, "\n\n"), "\n")
			if rest == "" {
				return os.WriteFile(path, nil, 0o644)
			}
			return os.WriteFile(path, []byte(rest+"\n"), 0o644)
		},
	}, nil
}

func settingsStep(path string, plugins bool) (*step, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	var what []string
	if s, _ := m["outputStyle"].(string); strings.EqualFold(s, "gentleman") {
		what = append(what, "unset the Gentleman output style")
	}
	if plugins {
		if p, ok := m["enabledPlugins"].(map[string]any); ok && p["engram@engram"] == true {
			what = append(what, "disable the Engram plugin")
		}
	}
	if len(what) == 0 {
		return nil, nil
	}
	return &step{
		desc:  fmt.Sprintf("claude: %s in %s (the file is rewritten with sorted keys)", strings.Join(what, " and "), path),
		paths: []string{path},
		run: func() error {
			if s, _ := m["outputStyle"].(string); strings.EqualFold(s, "gentleman") {
				delete(m, "outputStyle")
			}
			if plugins {
				if p, ok := m["enabledPlugins"].(map[string]any); ok && p["engram@engram"] == true {
					p["engram@engram"] = false
				}
			}
			b, err := json.MarshalIndent(m, "", "  ")
			if err != nil {
				return err
			}
			return os.WriteFile(path, append(b, '\n'), 0o644)
		},
	}, nil
}

// sddPaths lists the workflow files Gentle-AI installs. Generic skills such as
// branch-pr or go-testing are not listed: they are useful on their own.
func sddPaths(home string) []string {
	patterns := []string{
		filepath.Join(home, ".claude", "skills", "sdd-*"),
		filepath.Join(home, ".claude", "skills", "_shared"),
		filepath.Join(home, ".claude", "agents", "sdd-*.md"),
		filepath.Join(home, ".claude", "commands", "sdd-*.md"),
		filepath.Join(home, ".claude", "output-styles", "gentleman.md"),
		filepath.Join(home, ".cursor", "skills", "sdd-*"),
		filepath.Join(home, ".cursor", "skills", "_shared"),
		filepath.Join(home, ".cursor", "agents", "sdd-*.md"),
	}
	var out []string
	for _, p := range patterns {
		m, _ := filepath.Glob(p)
		out = append(out, m...)
	}
	return out
}
