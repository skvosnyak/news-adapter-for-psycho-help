package news

import (
	"sync"
	"time"
)

type Store struct {
	mu         sync.RWMutex
	items      []News
	bySlug     map[string]News
	bySourceID map[int]News
	updatedAt  time.Time
}

func NewStore() *Store {
	return &Store{items: []News{}, bySlug: map[string]News{}, bySourceID: map[int]News{}}
}

func (s *Store) Replace(items []News) {
	if items == nil {
		items = []News{}
	}
	bySlug := make(map[string]News, len(items))
	bySourceID := make(map[int]News, len(items))
	for _, n := range items {
		bySlug[n.ID] = n
		bySourceID[n.SourceID] = n
	}

	s.mu.Lock()
	s.items = items
	s.bySlug = bySlug
	s.bySourceID = bySourceID
	s.updatedAt = time.Now().UTC()
	s.mu.Unlock()
}

// List возвращает все новости (от новых к старым) и время последнего обновления.
// Срез отдаётся без копирования: Replace никогда не меняет старый срез,
// а подменяет его новым, поэтому читать его безопасно. Изменять — нельзя.
func (s *Store) List() ([]News, time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.items, s.updatedAt
}

// Get ищет новость по slug. Если точного совпадения нет (пост отредактировали,
// и заголовок, а с ним slug, изменился), ищет по номеру в конце slug'а.
func (s *Store) Get(slug string) (News, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if n, ok := s.bySlug[slug]; ok {
		return n, true
	}
	if id, ok := SourceIDFromSlug(slug); ok {
		n, ok := s.bySourceID[id]
		return n, ok
	}
	return News{}, false
}

// Ready сообщает, была ли хотя бы одна успешная загрузка.
func (s *Store) Ready() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.updatedAt.IsZero()
}
