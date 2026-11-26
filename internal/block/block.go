// Package block edits named, marked sections inside a text file, so a tool
// can install, update and remove its own part without touching the rest.
package block

import "strings"

func open(prefix, name string) string    { return "<!-- " + prefix + ":" + name + " -->" }
func closing(prefix, name string) string { return "<!-- /" + prefix + ":" + name + " -->" }

// Find returns the byte range of the named block, markers included, plus the
// newline that follows the closing marker.
func Find(text, prefix, name string) (start, end int, ok bool) {
	o := open(prefix, name)
	s := strings.Index(text, o)
	if s < 0 {
		return 0, 0, false
	}
	c := closing(prefix, name)
	rel := strings.Index(text[s+len(o):], c)
	if rel < 0 {
		return 0, 0, false
	}
	e := s + len(o) + rel + len(c)
	if e < len(text) && text[e] == '\n' {
		e++
	}
	return s, e, true
}

// Wrap puts body between the markers.
func Wrap(prefix, name, body string) string {
	return open(prefix, name) + "\n" + strings.TrimRight(body, "\n") + "\n" + closing(prefix, name) + "\n"
}

// Upsert replaces the named block, or appends it after a blank line.
func Upsert(text, prefix, name, body string) string {
	b := Wrap(prefix, name, body)
	if s, e, ok := Find(text, prefix, name); ok {
		return text[:s] + b + text[e:]
	}
	if text == "" {
		return b
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return text + "\n" + b
}

// Remove deletes the named block. When the block was appended at the end it
// also drops the blank line Upsert added, so install then remove returns the
// original text.
func Remove(text, prefix, name string) (string, bool) {
	s, e, ok := Find(text, prefix, name)
	if !ok {
		return text, false
	}
	if e == len(text) && s >= 2 && text[s-1] == '\n' && text[s-2] == '\n' {
		s--
	}
	return text[:s] + text[e:], true
}

// Names lists the block names present for a prefix, in order of appearance.
func Names(text, prefix string) []string {
	needle := "<!-- " + prefix + ":"
	var out []string
	rest := text
	for {
		i := strings.Index(rest, needle)
		if i < 0 {
			return out
		}
		rest = rest[i+len(needle):]
		j := strings.Index(rest, " -->")
		if j < 0 {
			return out
		}
		if name := rest[:j]; name != "" && !strings.ContainsAny(name, "\n ") {
			out = append(out, name)
		}
		rest = rest[j:]
	}
}
