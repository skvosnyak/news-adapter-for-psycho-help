// Package news — «наша» модель новости и всё, что не зависит от VK:
// хранилище в памяти и фоновое обновление.
package news

import (
	"context"
	"time"
)

// News — формат новости для фронта. Соответствует TS-типу:
//
//	type TNews = {
//	  id: string;              // slug
//	  images: string[] | null; // у нас всегда массив, пустой если картинок нет
//	  type: TNewsType | null;  // в VK типов нет → всегда null
//	  date: string;            // ISO 8601
//	  title: string;
//	  text: string | null;
//	};
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

// Source — откуда берутся новости. Сейчас это VK, но остальной код
// об этом не знает: ему нужен только метод Fetch.
type Source interface {
	Fetch(ctx context.Context) ([]News, error)
}
