package block

import (
	"reflect"
	"testing"
)

func TestInstallThenRemoveRestoresOriginal(t *testing.T) {
	for _, orig := range []string{"", "# Mine\n\nhello\n", "# Mine\n\nhello\n\n\nbye\n"} {
		got := Upsert(orig, "rubio-ia", "core", "body line")
		if orig != "" && got == orig {
			t.Fatalf("upsert changed nothing for %q", orig)
		}
		back, ok := Remove(got, "rubio-ia", "core")
		if !ok || back != orig {
			t.Errorf("round trip of %q gave %q (ok=%v)", orig, back, ok)
		}
	}
}

func TestUpsertIsIdempotentAndReplaces(t *testing.T) {
	once := Upsert("user text\n", "rubio-ia", "core", "v1")
	twice := Upsert(once, "rubio-ia", "core", "v1")
	if once != twice {
		t.Error("second upsert with the same body must not change the text")
	}
	updated := Upsert(twice, "rubio-ia", "core", "v2")
	if updated == twice || !contains(updated, "v2") || contains(updated, "v1") {
		t.Errorf("block not replaced in place: %q", updated)
	}
	if !contains(updated, "user text") {
		t.Error("user text was lost")
	}
}

func TestBlockInTheMiddleKeepsNeighbours(t *testing.T) {
	text := "before\n" + Wrap("x", "a", "A") + "after\n"
	got, ok := Remove(text, "x", "a")
	if !ok || got != "before\nafter\n" {
		t.Errorf("got %q ok=%v", got, ok)
	}
}

func TestMissingOrUnclosedBlock(t *testing.T) {
	if _, ok := Remove("nothing here", "x", "a"); ok {
		t.Error("remove must report not found")
	}
	if _, _, ok := Find("<!-- x:a -->\nno end", "x", "a"); ok {
		t.Error("an unclosed block must not be found")
	}
}

func TestNames(t *testing.T) {
	text := "<!-- gentle-ai:persona -->\na\n<!-- /gentle-ai:persona -->\n<!-- other:x -->\n<!-- gentle-ai:sdd-orchestrator -->\n"
	got := Names(text, "gentle-ai")
	want := []string{"persona", "sdd-orchestrator"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
