package news

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// fakeSource отдаёт заранее заданный результат и считает вызовы.
type fakeSource struct {
	items []News
	err   error
	calls atomic.Int32
}

func (f *fakeSource) Fetch(
	context.Context,
) (
	[]News,
	error,
) {
	f.calls.Add(1)
	return f.items, f.err
}

func newTestUpdater(
	src Source,
	store *Store,
	file string,
) *Updater {
	return NewUpdater(src, store, file, time.Hour, time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestSaveLoadFileRestoresSourceID(
	t *testing.T,
) {
	path := filepath.Join(t.TempDir(), "news.json")
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	if err := SaveFile(path, []News{{ID: "khakaton-7", SourceID: 7, Title: "Хакатон"}}, at); err != nil {
		t.Fatal(err)
	}

	items, updatedAt, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !updatedAt.Equal(at) || len(items) != 1 || items[0].SourceID != 7 || items[0].Title != "Хакатон" {
		t.Errorf("unexpected load: %v %+v", updatedAt, items)
	}
}

func TestRefreshSavesFileAndStore(
	t *testing.T,
) {
	path := filepath.Join(t.TempDir(), "store", "news.json")
	store := NewStore()
	u := newTestUpdater(&fakeSource{items: []News{{ID: "a-1", SourceID: 1}}}, store, path)

	if err := u.refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if items, _ := store.List(); len(items) != 1 {
		t.Errorf("store not updated: %+v", items)
	}
	if items, _, err := LoadFile(path); err != nil || len(items) != 1 {
		t.Errorf("file not saved: %v %+v", err, items)
	}
}

func TestSourceErrorKeepsSavedNews(
	t *testing.T,
) {
	path := filepath.Join(t.TempDir(), "news.json")
	saved := time.Now().Add(-2 * time.Hour).UTC() // старше интервала — сразу пойдёт в источник
	if err := SaveFile(path, []News{{ID: "a-1", SourceID: 1}}, saved); err != nil {
		t.Fatal(err)
	}

	for name, src := range map[string]*fakeSource{
		"vk error":       {err: errors.New("vk api error 5: User authorization failed")},
		"empty response": {items: []News{}},
	} {
		t.Run(name, func(
			t *testing.T,
		) {
			store := NewStore()
			u := newTestUpdater(src, store, path)

			if wait := u.loadFile(); wait != 0 {
				t.Errorf("stale file must trigger sync right away, got wait %v", wait)
			}
			if err := u.refresh(context.Background()); err == nil {
				t.Fatal("want error")
			}

			if items, _ := store.List(); !store.Ready() || len(items) != 1 {
				t.Errorf("saved news must still be served: %+v", items)
			}
			if items, at, _ := LoadFile(path); len(items) != 1 || !at.Equal(saved) {
				t.Errorf("file must stay untouched: %v %+v", at, items)
			}
		})
	}
}

func TestFreshFileDelaysFirstSync(
	t *testing.T,
) {
	path := filepath.Join(t.TempDir(), "news.json")
	if err := SaveFile(path, []News{{ID: "a-1", SourceID: 1}}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	u := newTestUpdater(&fakeSource{}, NewStore(), path)
	if wait := u.loadFile(); wait < 59*time.Minute {
		t.Errorf("fresh file: want to wait ~1h before sync, got %v", wait)
	}
}

func TestMissingFileSyncsImmediately(
	t *testing.T,
) {
	store := NewStore()
	u := newTestUpdater(&fakeSource{}, store, filepath.Join(t.TempDir(), "news.json"))
	if wait := u.loadFile(); wait != 0 || store.Ready() {
		t.Errorf("no file: want immediate sync and empty store, got wait %v ready %v", wait, store.Ready())
	}
}
