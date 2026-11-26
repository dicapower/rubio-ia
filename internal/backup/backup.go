// Package backup keeps a copy of exactly the files a command is about to
// change, with a manifest, so the change can be undone. It never copies
// anything else: no history, no databases.
package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// keep is how many backups stay on disk. They are small, but not infinite.
const keep = 10

// Entry records one path: whether it existed before the change.
type Entry struct {
	Path    string `json:"path"`
	Existed bool   `json:"existed"`
	Dir     bool   `json:"dir,omitempty"`
}

// Manifest describes one backup.
type Manifest struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Entries   []Entry   `json:"entries"`
}

// Session collects the paths of one command.
type Session struct {
	root string
	m    Manifest
	seen map[string]bool
}

func rootDir(home string) string { return filepath.Join(home, ".rubio-ia", "backups") }

// Begin starts a backup session. Nothing is written until the first Add.
func Begin(home string) *Session {
	now := time.Now().UTC()
	return &Session{
		root: rootDir(home),
		m: Manifest{
			ID:        fmt.Sprintf("%s-%03d", now.Format("20060102T150405"), now.Nanosecond()/1e6),
			CreatedAt: now,
		},
		seen: map[string]bool{},
	}
}

// ID is the backup identifier, valid after Begin.
func (s *Session) ID() string { return s.m.ID }

func snapRel(path string) string {
	p := filepath.ToSlash(path)
	p = strings.ReplaceAll(p, ":", "")
	return strings.TrimPrefix(p, "/")
}

func (s *Session) snapPath(path string) string {
	return filepath.Join(s.root, s.m.ID, "files", filepath.FromSlash(snapRel(path)))
}

// Add snapshots a file or directory before it is changed. A path that does not
// exist is recorded as absent, so restore can remove what a command created.
func (s *Session) Add(path string) error {
	path = filepath.Clean(path)
	if s.seen[path] {
		return nil
	}
	for p := filepath.Dir(path); p != filepath.Dir(p); p = filepath.Dir(p) {
		if s.seen[p] {
			return nil // an ancestor directory is already saved
		}
	}
	s.seen[path] = true
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		s.m.Entries = append(s.m.Entries, Entry{Path: path})
		return nil
	}
	if err != nil {
		return fmt.Errorf("backup %s: %w", path, err)
	}
	e := Entry{Path: path, Existed: true, Dir: info.IsDir()}
	if err := copyTree(path, s.snapPath(path)); err != nil {
		return fmt.Errorf("backup %s: %w", path, err)
	}
	s.m.Entries = append(s.m.Entries, e)
	return nil
}

// Commit writes the manifest and prunes old backups. A session with no entries
// leaves nothing behind.
func (s *Session) Commit() error {
	if len(s.m.Entries) == 0 {
		return nil
	}
	dir := filepath.Join(s.root, s.m.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}
	raw, err := json.MarshalIndent(s.m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return prune(s.root)
}

func prune(root string) error {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var ids []string
	for _, e := range ents {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	sort.Strings(ids)
	for len(ids) > keep {
		if err := os.RemoveAll(filepath.Join(root, ids[0])); err != nil {
			return fmt.Errorf("prune backup %s: %w", ids[0], err)
		}
		ids = ids[1:]
	}
	return nil
}

// List returns the backups, newest first.
func List(home string) ([]Manifest, error) {
	root := rootDir(home)
	ents, err := os.ReadDir(root)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read backups: %w", err)
	}
	var out []Manifest
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, e.Name(), "manifest.json"))
		if err != nil {
			continue
		}
		var m Manifest
		if json.Unmarshal(raw, &m) == nil {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

// Restore puts every path of a backup back as it was. An empty id means the
// newest backup.
func Restore(home, id string) (Manifest, error) {
	all, err := List(home)
	if err != nil {
		return Manifest{}, err
	}
	if len(all) == 0 {
		return Manifest{}, errors.New("no backups found")
	}
	m := all[0]
	if id != "" {
		found := false
		for _, c := range all {
			if c.ID == id {
				m, found = c, true
			}
		}
		if !found {
			return Manifest{}, fmt.Errorf("backup %q not found", id)
		}
	}
	s := &Session{root: rootDir(home), m: m}
	for i := len(m.Entries) - 1; i >= 0; i-- {
		e := m.Entries[i]
		if err := os.RemoveAll(e.Path); err != nil {
			return m, fmt.Errorf("restore %s: %w", e.Path, err)
		}
		if !e.Existed {
			continue
		}
		if err := copyTree(s.snapPath(e.Path), e.Path); err != nil {
			return m, fmt.Errorf("restore %s: %w", e.Path, err)
		}
	}
	return m, nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o700)
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return err
			}
			return os.Symlink(link, target)
		default:
			return copyFile(p, target, info.Mode().Perm())
		}
	})
}

func copyFile(src, dst string, perm fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
