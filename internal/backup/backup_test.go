package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRestoreUndoesChangeCreationAndDeletion(t *testing.T) {
	home := t.TempDir()
	changed := filepath.Join(home, "a.md")
	deleted := filepath.Join(home, "dir", "b.md")
	created := filepath.Join(home, "new.md")
	write(t, changed, "original")
	write(t, deleted, "keep me")

	s := Begin(home)
	for _, p := range []string{changed, deleted, created} {
		if err := s.Add(p); err != nil {
			t.Fatal(err)
		}
	}
	write(t, changed, "modified")
	write(t, created, "made by the command")
	if err := os.Remove(deleted); err != nil {
		t.Fatal(err)
	}
	if err := s.Commit(); err != nil {
		t.Fatal(err)
	}

	if _, err := Restore(home, ""); err != nil {
		t.Fatal(err)
	}
	if got := read(t, changed); got != "original" {
		t.Errorf("changed file: %q", got)
	}
	if got := read(t, deleted); got != "keep me" {
		t.Errorf("deleted file: %q", got)
	}
	if _, err := os.Stat(created); !os.IsNotExist(err) {
		t.Error("a file the command created must be removed on restore")
	}
}

func TestDirectoryBackupAndAncestorSkip(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "skills")
	write(t, filepath.Join(dir, "x", "SKILL.md"), "x")
	s := Begin(home)
	if err := s.Add(dir); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(filepath.Join(dir, "x", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if len(s.m.Entries) != 1 {
		t.Fatalf("a file inside a saved dir must not be saved twice: %d entries", len(s.m.Entries))
	}
	if err := s.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := Restore(home, s.ID()); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(dir, "x", "SKILL.md")); got != "x" {
		t.Errorf("got %q", got)
	}
}

func TestEmptySessionLeavesNothingAndOldBackupsArePruned(t *testing.T) {
	home := t.TempDir()
	if err := Begin(home).Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(rootDir(home)); !os.IsNotExist(err) {
		t.Error("an empty session must not create the backup folder")
	}
	f := filepath.Join(home, "f")
	write(t, f, "v")
	for i := 0; i < keep+3; i++ {
		s := Begin(home)
		s.m.ID = s.m.ID + string(rune('a'+i)) // distinct ids within one millisecond
		if err := s.Add(f); err != nil {
			t.Fatal(err)
		}
		if err := s.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	all, err := List(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != keep {
		t.Errorf("kept %d backups, want %d", len(all), keep)
	}
}

func TestRestoreWithoutBackups(t *testing.T) {
	if _, err := Restore(t.TempDir(), ""); err == nil {
		t.Fatal("expected an error")
	}
}
