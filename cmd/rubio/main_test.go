package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEndToEnd(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	do := func(args ...string) (string, int) {
		var out, errw bytes.Buffer
		code := run(args, home, &out, &errw)
		return out.String() + errw.String(), code
	}

	if o, c := do("setup", "--lang", "es-ES"); c != 0 || !strings.Contains(o, "core written") {
		t.Fatalf("setup: %d %s", c, o)
	}
	if o, _ := do("doctor"); !strings.Contains(o, "core installed") {
		t.Errorf("doctor: %s", o)
	}
	if o, _ := do("restore", "--list"); strings.Contains(o, "No backups") {
		t.Errorf("list: %s", o)
	}
	if o, c := do("remove"); c != 0 || !strings.Contains(o, "core removed") {
		t.Fatalf("remove: %d %s", c, o)
	}
	if o, c := do("restore"); c != 0 || !strings.Contains(o, "Restored") {
		t.Fatalf("restore: %d %s", c, o)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "CLAUDE.md")); err != nil {
		t.Error("restore did not bring back the installed file")
	}
}

func TestBadUsage(t *testing.T) {
	var out, errw bytes.Buffer
	if run(nil, t.TempDir(), &out, &errw) != 2 {
		t.Error("no args must exit 2")
	}
	if run([]string{"nope"}, t.TempDir(), &out, &errw) != 2 {
		t.Error("unknown command must exit 2")
	}
	if run([]string{"setup", "--bogus"}, t.TempDir(), &out, &errw) != 1 {
		t.Error("bad flag must exit 1")
	}
	if run([]string{"migrate"}, t.TempDir(), &out, &errw) != 1 {
		t.Error("migrate without a target must exit 1")
	}
	out.Reset()
	if run([]string{"version"}, t.TempDir(), &out, &errw) != 0 || !strings.Contains(out.String(), "rubio") {
		t.Error("version")
	}
}
