package news

import (
	"context"
	"time"
)

type News struct {
	ID     string    `json:"id"`
	Images []string  `json:"images"`
	Type   *string   `json:"type"` // 'Анонс мероприятия' | 'Отчет о мероприятии' | null
	Date   time.Time `json:"date"`
	Title  string    `json:"title"`
	Text   *string   `json:"text"`

	// SourceID — номер поста в источнике (в VK). Наружу не отдаётся,
	// нужен, чтобы найти новость, даже если её slug поменялся.
	SourceID int `json:"-"`
}

type Source interface {
	Fetch(ctx context.Context) ([]News, error)
}
