package vk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Фейковый ответ wall.get: закреп (старый), пост с фото во всех размерах,
// пост только с фото без текста, реклама и репост.
const sampleWall = `{"response":{"count":5,"items":[
 {"id":1,"owner_id":-42,"date":1700000000,"is_pinned":1,"text":"Старый закреп"},
 {"id":5,"owner_id":-42,"date":1760000000,"text":"День открытых дверей в [club42|Политехе]!\nПриходите 1 ноября.",
  "attachments":[{"type":"photo","photo":{"sizes":[
    {"type":"s","url":"https://img/s","width":75,"height":50},
    {"type":"r","url":"https://img/r-crop","width":510,"height":340},
    {"type":"z","url":"https://img/z","width":1080,"height":720},
    {"type":"w","url":"https://img/w","width":2560,"height":1707}]}},
   {"type":"video","video":{}}]},
 {"id":4,"owner_id":-42,"date":1750000000,"text":"",
  "attachments":[{"type":"photo","photo":{"sizes":[{"type":"z","url":"https://img/only"}]}}]},
 {"id":6,"owner_id":-42,"date":1760000100,"marked_as_ads":1,"text":"Реклама"},
 {"id":7,"owner_id":-42,"date":1760000200,"text":"Репост","copy_history":[{"id":1}]}
]}}`

func newTestClient(t *testing.T, body string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil || r.PostForm.Get("access_token") != "secret" {
			t.Errorf("token must be sent in POST form")
		}
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	c := NewClient("secret", "5.199", "spp_polytech", 100)
	c.BaseURL = srv.URL
	return c
}

func TestFetchConvertsAndFilters(t *testing.T) {
	items, err := newTestClient(t, sampleWall).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 news (ads, repost, no-text filtered), got %d", len(items))
	}

	n := items[0] // самый новый, хотя закреп в VK был первым
	if n.ID != "den-otkrytykh-dverey-v-politekhe-5" {
		t.Errorf("bad slug %q", n.ID)
	}
	if n.Title != "День открытых дверей в Политехе!" || n.Text == nil || *n.Text != "Приходите 1 ноября." {
		t.Errorf("bad title/text: %q / %v", n.Title, n.Text)
	}
	if len(n.Images) != 1 || n.Images[0] != "https://img/z" {
		t.Errorf("want one z-size photo, got %v", n.Images)
	}

	// Проверяем JSON целиком на совпадение с TS-типом TNews.
	b, _ := json.Marshal(items[1])
	want := `{"id":"staryy-zakrep-1","images":[],"type":null,"date":"2023-11-14T22:13:20Z","title":"Старый закреп","text":null}`
	if string(b) != want {
		t.Errorf("json mismatch:\n got %s\nwant %s", b, want)
	}
}

func TestVKErrorWith200(t *testing.T) {
	c := newTestClient(t, `{"error":{"error_code":5,"error_msg":"User authorization failed"}}`)
	_, err := c.Fetch(context.Background())
	if err == nil || !strings.Contains(err.Error(), "User authorization failed") {
		t.Fatalf("want vk error, got %v", err)
	}
}

func TestLongFirstLineBecomesTruncatedTitle(t *testing.T) {
	long := strings.Repeat("слово ", 40) // 240 символов, одна строка
	n, _ := convertPost(Post{ID: 1, Text: long})
	if !strings.HasSuffix(n.Title, "слово...") || len([]rune(n.Title)) > maxTitleRunes+3 {
		t.Errorf("bad title %q", n.Title)
	}
	if n.Text == nil || *n.Text != strings.TrimSpace(long) {
		t.Error("full text must be kept in text")
	}
}
