// Package install writes and removes the Rubio-IA instructions. Every change
// is backed up first and every command can run as a dry run.
package install

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dicapower/rubio-ia/internal/agent"
	"github.com/dicapower/rubio-ia/internal/assets"
	"github.com/dicapower/rubio-ia/internal/backup"
	"github.com/dicapower/rubio-ia/internal/block"
	"github.com/dicapower/rubio-ia/internal/tokens"
)

// Options for Setup.
type Options struct {
	Agents []string // empty or "all" means every agent that is installed
	Lang   string
	Skills bool
	DryRun bool
}

// Resolve turns agent names into installed agents.
func Resolve(home string, names []string) ([]agent.Agent, error) {
	if len(names) == 0 || (len(names) == 1 && names[0] == "all") {
		var out []agent.Agent
		for _, a := range agent.All(home) {
			if a.Detected() {
				out = append(out, a)
			}
		}
		if len(out) == 0 {
			return nil, errors.New("no supported agent found (looked for ~/.claude and ~/.cursor)")
		}
		return out, nil
	}
	var out []agent.Agent
	for _, n := range names {
		a, err := agent.New(strings.TrimSpace(n), home)
		if err != nil {
			return nil, err
		}
		if !a.Detected() {
			return nil, fmt.Errorf("%s not found (%s does not exist)", a.Name, a.Dir())
		}
		out = append(out, a)
	}
	return out, nil
}

// Setup installs the core and, optionally, the skills.
func Setup(home string, o Options, out io.Writer) error {
	core, err := assets.Core(o.Lang)
	if err != nil {
		return err
	}
	skills, err := assets.Skills()
	if err != nil {
		return err
	}
	ags, err := Resolve(home, o.Agents)
	if err != nil {
		return err
	}
	sess := backup.Begin(home)
	changed := false
	gentle := false

	for _, a := range ags {
		path := a.Instructions()
		old, err := readOptional(path)
		if err != nil {
			return err
		}
		neu := a.Compose(old, core)
		switch {
		case neu == old:
			fmt.Fprintf(out, "%s: core already up to date (~%d tokens)\n", a.Name, tokens.Estimate(core))
		case o.DryRun:
			fmt.Fprintf(out, "%s: would write the core to %s (~%d tokens)\n", a.Name, path, tokens.Estimate(core))
		default:
			if err := sess.Add(path); err != nil {
				return err
			}
			if err := writeFile(path, neu); err != nil {
				return err
			}
			changed = true
			fmt.Fprintf(out, "%s: core written to %s (~%d tokens)\n", a.Name, path, tokens.Estimate(core))
		}
		if len(block.Names(neu, "gentle-ai")) > 0 {
			gentle = true
		}

		if !o.Skills {
			continue
		}
		for name, content := range skills {
			dir := filepath.Join(a.SkillsDir(), name)
			file := filepath.Join(dir, "SKILL.md")
			cur, err := readOptional(file)
			if err != nil {
				return err
			}
			switch {
			case cur != "" && !strings.Contains(cur, assets.SkillMarker):
				fmt.Fprintf(out, "%s: skill %s skipped, %s exists and is not from Rubio-IA\n", a.Name, name, file)
			case cur == content:
				fmt.Fprintf(out, "%s: skill %s already up to date\n", a.Name, name)
			case o.DryRun:
				fmt.Fprintf(out, "%s: would write skill %s\n", a.Name, name)
			default:
				if err := sess.Add(dir); err != nil {
					return err
				}
				if err := writeFile(file, content); err != nil {
					return err
				}
				changed = true
				fmt.Fprintf(out, "%s: skill %s written\n", a.Name, name)
			}
		}
	}

	if changed {
		if err := sess.Commit(); err != nil {
			return err
		}
		fmt.Fprintf(out, "Backup %s (undo with: rubio restore %s)\n", sess.ID(), sess.ID())
		fmt.Fprintln(out, "Restart your agent so it loads the new instructions, then run: rubio doctor")
	}
	if gentle {
		fmt.Fprintln(out, "Note: Gentle-AI instructions are still installed and will compete with these. See: rubio migrate gentle-ai")
	}
	return nil
}

// Remove takes Rubio-IA out of the given agents.
func Remove(home string, names []string, dryRun bool, out io.Writer) error {
	ags, err := Resolve(home, names)
	if err != nil {
		return err
	}
	skills, err := assets.Skills()
	if err != nil {
		return err
	}
	sess := backup.Begin(home)
	changed := false
	for _, a := range ags {
		path := a.Instructions()
		old, err := readOptional(path)
		if err != nil {
			return err
		}
		if rest, ok := a.Strip(old); !ok {
			fmt.Fprintf(out, "%s: no Rubio-IA core in %s\n", a.Name, path)
		} else if dryRun {
			fmt.Fprintf(out, "%s: would remove the core from %s\n", a.Name, path)
		} else {
			if err := sess.Add(path); err != nil {
				return err
			}
			if rest == "" {
				err = os.Remove(path)
			} else {
				err = writeFile(path, rest)
			}
			if err != nil {
				return fmt.Errorf("update %s: %w", path, err)
			}
			changed = true
			fmt.Fprintf(out, "%s: core removed from %s\n", a.Name, path)
		}

		for name := range skills {
			dir := filepath.Join(a.SkillsDir(), name)
			cur, err := readOptional(filepath.Join(dir, "SKILL.md"))
			if err != nil {
				return err
			}
			if !strings.Contains(cur, assets.SkillMarker) {
				continue
			}
			if dryRun {
				fmt.Fprintf(out, "%s: would remove skill %s\n", a.Name, name)
				continue
			}
			if err := sess.Add(dir); err != nil {
				return err
			}
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("remove %s: %w", dir, err)
			}
			changed = true
			fmt.Fprintf(out, "%s: skill %s removed\n", a.Name, name)
		}
	}
	if changed {
		if err := sess.Commit(); err != nil {
			return err
		}
		fmt.Fprintf(out, "Backup %s (undo with: rubio restore %s)\n", sess.ID(), sess.ID())
	}
	return nil
}

func readOptional(path string) (string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(b), nil
}

// writeFile keeps the file's permissions when it already exists.
func writeFile(path, content string) error {
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
