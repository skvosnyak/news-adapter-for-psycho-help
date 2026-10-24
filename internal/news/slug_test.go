package news

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"День открытых дверей 2026!": "den-otkrytykh-dverey-2026-512",
		"📢 Анонс: хакатон «CTF»":     "anons-khakaton-ctf-512",
		"!!!": "512",
	}
	for title, want := range cases {
		if got := Slugify(title, 512); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", title, got, want)
		}
	}
	long := Slugify(strings.Repeat("очень длинный заголовок ", 10), 1)
	if len(long) > maxSlugBaseRunes+2 || strings.Contains(long, "--") {
		t.Errorf("bad long slug %q", long)
	}
}

func TestStoreGetFallsBackToSourceID(t *testing.T) {
	s := NewStore()
	s.Replace([]News{{ID: "novyy-zagolovok-7", SourceID: 7}})
	if _, ok := s.Get("novyy-zagolovok-7"); !ok {
		t.Error("exact slug must match")
	}
	if _, ok := s.Get("staryy-zagolovok-7"); !ok {
		t.Error("old slug with same post number must still resolve")
	}
	if _, ok := s.Get("chto-to-8"); ok {
		t.Error("unknown post must not resolve")
	}
}
