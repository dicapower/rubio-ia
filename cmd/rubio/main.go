// Command rubio installs and maintains the Rubio-IA working agreement for
// coding agents.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dicapower/rubio-ia/internal/assets"
	"github.com/dicapower/rubio-ia/internal/backup"
	"github.com/dicapower/rubio-ia/internal/doctor"
	"github.com/dicapower/rubio-ia/internal/install"
	"github.com/dicapower/rubio-ia/internal/migrate"
)

// version is set at release time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `rubio %s: a short working agreement for your coding agent.

Usage:
  rubio setup [--agent claude|cursor|all] [--lang auto|es-ES|en] [--no-skills] [--dry-run]
  rubio remove [--agent claude|cursor|all] [--dry-run]
  rubio doctor
  rubio restore [ID]        undo the last change (or the one named); --list shows backups
  rubio migrate gentle-ai   [--yes] [--skills] [--plugins]
  rubio version

Every change is backed up first. Nothing leaves your machine.
`

func main() {
	home, err := os.UserHomeDir()
	if h := os.Getenv("RUBIO_HOME"); h != "" {
		home, err = h, nil // lets tests and cautious users point at another folder
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot find your home folder:", err)
		os.Exit(1)
	}
	os.Exit(run(os.Args[1:], home, os.Stdout, os.Stderr))
}

func run(args []string, home string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintf(errw, usage, version)
		return 2
	}
	var err error
	switch args[0] {
	case "setup":
		err = cmdSetup(args[1:], home, out)
	case "remove":
		err = cmdRemove(args[1:], home, out)
	case "doctor":
		doctor.Print(out, doctor.Run(home))
	case "restore":
		err = cmdRestore(args[1:], home, out)
	case "migrate":
		err = cmdMigrate(args[1:], home, out)
	case "version", "--version", "-v":
		fmt.Fprintln(out, "rubio", version)
	case "help", "--help", "-h":
		fmt.Fprintf(out, usage, version)
	default:
		fmt.Fprintf(errw, "unknown command %q\n\n"+usage, args[0], version)
		return 2
	}
	if err != nil {
		fmt.Fprintln(errw, "error:", err)
		return 1
	}
	return 0
}

func newFlags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func splitAgents(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func cmdSetup(args []string, home string, out io.Writer) error {
	fs := newFlags("setup")
	agent := fs.String("agent", "all", "agents to configure, comma separated")
	lang := fs.String("lang", "auto", "language rule: "+strings.Join(assets.Languages(), ", "))
	noSkills := fs.Bool("no-skills", false, "install only the core, not the rubio-plan skill")
	dry := fs.Bool("dry-run", false, "show what would change")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return install.Setup(home, install.Options{
		Agents: splitAgents(*agent), Lang: *lang, Skills: !*noSkills, DryRun: *dry,
	}, out)
}

func cmdRemove(args []string, home string, out io.Writer) error {
	fs := newFlags("remove")
	agent := fs.String("agent", "all", "agents to clean, comma separated")
	dry := fs.Bool("dry-run", false, "show what would change")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return install.Remove(home, splitAgents(*agent), *dry, out)
}

func cmdRestore(args []string, home string, out io.Writer) error {
	fs := newFlags("restore")
	list := fs.Bool("list", false, "list backups")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *list {
		all, err := backup.List(home)
		if err != nil {
			return err
		}
		if len(all) == 0 {
			fmt.Fprintln(out, "No backups yet.")
		}
		for _, m := range all {
			fmt.Fprintf(out, "%s  %d paths\n", m.ID, len(m.Entries))
		}
		return nil
	}
	id := ""
	if fs.NArg() > 0 {
		id = fs.Arg(0)
	}
	m, err := backup.Restore(home, id)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Restored backup %s (%d paths).\n", m.ID, len(m.Entries))
	return nil
}

func cmdMigrate(args []string, home string, out io.Writer) error {
	if len(args) == 0 || args[0] != "gentle-ai" {
		return fmt.Errorf("usage: rubio migrate gentle-ai [--yes] [--skills] [--plugins]")
	}
	fs := newFlags("migrate")
	yes := fs.Bool("yes", false, "apply the plan (default: only print it)")
	skills := fs.Bool("skills", false, "also remove the SDD skills, agents, commands and output style")
	plugins := fs.Bool("plugins", false, "also disable the Engram plugin in Claude Code")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	return migrate.GentleAI(home, migrate.Options{Yes: *yes, Skills: *skills, Plugins: *plugins}, out)
}
