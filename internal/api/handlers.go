// Package api — HTTP-эндпоинты, которые вызывает фронт.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"news-service/internal/news"
)

const (
	defaultLimit = 10
	maxLimit     = 50
)

type handler struct {
	store *news.Store
}

// NewRouter регистрирует маршруты. Шаблоны вида "GET /api/news/{id}"
// понимает стандартный http.ServeMux начиная с Go 1.22.
func NewRouter(store *news.Store, corsOrigin string) http.Handler {
	h := &handler{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/news", h.list)
	mux.HandleFunc("GET /api/news/{slug}", h.get)
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /api/docs", docs)
	mux.HandleFunc("GET /api/docs/openapi.yaml", spec)
	return withCORS(corsOrigin, mux)
}

type listResponse struct {
	Items     []news.News `json:"items"`
	Total     int         `json:"total"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// list — GET /api/news?limit=10&offset=0 — лента новостей с пагинацией.
func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	if !h.store.Ready() {
		writeError(w, http.StatusServiceUnavailable, "news are not loaded yet")
		return
	}

	limit, err := intQuery(r, "limit", defaultLimit)
	if err != nil || limit < 1 {
		writeError(w, http.StatusBadRequest, "invalid limit")
		return
	}
	limit = min(limit, maxLimit)

	offset, err := intQuery(r, "offset", 0)
	if err != nil || offset < 0 {
		writeError(w, http.StatusBadRequest, "invalid offset")
		return
	}

	items, updatedAt := h.store.List()
	start := min(offset, len(items))
	end := min(start+limit, len(items))

	writeJSON(w, http.StatusOK, listResponse{
		Items:     items[start:end],
		Total:     len(items),
		UpdatedAt: updatedAt,
	})
}

// get — GET /api/news/{slug} — одна новость (для отдельной страницы).
func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	if !h.store.Ready() {
		writeError(w, http.StatusServiceUnavailable, "news are not loaded yet")
		return
	}
	n, ok := h.store.Get(r.PathValue("slug"))
	if !ok {
		writeError(w, http.StatusNotFound, "news not found")
		return
	}
	writeJSON(w, http.StatusOK, n)
}

// health — GET /health — «жив ли сервис» для Docker/мониторинга.
func (h *handler) health(w http.ResponseWriter, r *http.Request) {
	_, updatedAt := h.store.List()
	resp := map[string]any{"status": "ok", "news_loaded": h.store.Ready()}
	if !updatedAt.IsZero() {
		resp["updated_at"] = updatedAt
	}
	writeJSON(w, http.StatusOK, resp)
}

// intQuery читает числовой query-параметр или возвращает значение по умолчанию.
func intQuery(r *http.Request, key string, def int) (int, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def, nil
	}
	return strconv.Atoi(v)
}

// writeJSON отправляет ответ в JSON с нужным статусом.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError — единый формат ошибок: {"error": "..."}.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// withCORS разрешает запросы с фронта на другом домене/порту.
// Если origin пустой (фронт и API за одним nginx) — ничего не делает.
func withCORS(origin string, next http.Handler) http.Handler {
	if origin == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
