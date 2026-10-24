package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"news-service/internal/news"
)

func TestListPaginationAndNotReady(t *testing.T) {
	store := news.NewStore()
	router := NewRouter(store, "")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/news", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503 before first load, got %d", rec.Code)
	}

	store.Replace([]news.News{{ID: "c-3"}, {ID: "b-2"}, {ID: "a-1"}})

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/news?limit=2&offset=1", nil))
	var resp listResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Total != 3 || len(resp.Items) != 2 || resp.Items[0].ID != "b-2" {
		t.Errorf("unexpected page: %+v", resp)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/news?offset=100", nil))
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &resp) != nil || resp.Items == nil {
		t.Errorf("offset past end must return empty items [], got %s", rec.Body)
	}
}

func TestGetBySlug(t *testing.T) {
	store := news.NewStore()
	store.Replace([]news.News{{ID: "khakaton-7", SourceID: 7, Title: "x"}})
	router := NewRouter(store, "")

	for path, want := range map[string]int{
		"/api/news/khakaton-7": 200,
		"/api/news/7":          200,
		"/api/news/drugoe-8":   404,
		"/api/news/abc":        404,
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != want {
			t.Errorf("%s: want %d, got %d", path, want, rec.Code)
		}
	}
}
