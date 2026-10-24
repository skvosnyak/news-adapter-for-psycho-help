package vk

import (
	"encoding/json"
	"fmt"
)

// apiResponse — общая обёртка любого ответа VK: либо response, либо error.
type apiResponse struct {
	Response *wallResponse `json:"response"`
	Error    *apiError     `json:"error"`
}

// apiError — ошибка VK (неверный токен, закрытая группа и т.п.).
type apiError struct {
	Code int    `json:"error_code"`
	Msg  string `json:"error_msg"`
}

func (e *apiError) Error() string {
	return fmt.Sprintf("vk api error %d: %s", e.Code, e.Msg)
}

// wallResponse — ответ метода wall.get.
type wallResponse struct {
	Count int    `json:"count"`
	Items []Post `json:"items"`
}

// Post — пост на стене. Описаны только нужные нам поля,
// остальное encoding/json просто пропустит.
type Post struct {
	ID          int               `json:"id"`
	OwnerID     int               `json:"owner_id"`
	Date        int64             `json:"date"` // unix-время
	Text        string            `json:"text"`
	IsPinned    int               `json:"is_pinned"`
	MarkedAsAds int               `json:"marked_as_ads"`
	CopyHistory []json.RawMessage `json:"copy_history"` // непустой = это репост
	Attachments []Attachment      `json:"attachments"`
}

// Attachment — вложение поста (фото, видео, ссылка...). Нас интересуют фото.
type Attachment struct {
	Type  string `json:"type"`
	Photo *Photo `json:"photo"`
}

// Photo — фотография; VK отдаёт её сразу в нескольких размерах.
type Photo struct {
	Sizes []PhotoSize `json:"sizes"`
}

// PhotoSize — один из размеров фотографии.
type PhotoSize struct {
	Type   string `json:"type"` // буква размера: s, m, x, y, z, w, o, p, q, r
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}
