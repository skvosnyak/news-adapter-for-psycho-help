// Package vk — всё, что знает про VK API: запрос к wall.get,
// структуры ответа и перевод постов в наш формат news.News.
package vk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"news-service/internal/news"
)

const defaultBaseURL = "https://api.vk.com/method"

// Client ходит в VK API. Реализует интерфейс news.Source.
type Client struct {
	BaseURL string // можно подменить в тестах
	token   string
	version string
	group   string
	count   int
	http    *http.Client
}

// NewClient создаёт клиента с таймаутом на запрос.
func NewClient(token, version, group string, count int) *Client {
	return &Client{
		BaseURL: defaultBaseURL,
		token:   token,
		version: version,
		group:   group,
		count:   count,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// Fetch — точка входа для Updater: получить посты и перевести их в новости.
func (c *Client) Fetch(ctx context.Context) ([]news.News, error) {
	posts, err := c.WallGet(ctx)
	if err != nil {
		return nil, err
	}
	return ToNews(posts), nil
}

// WallGet вызывает метод VK wall.get и возвращает сырые посты.
//
// Параметры отправляются POST-формой, а не в URL: так токен не попадёт
// в текст ошибки (net/http пишет URL в ошибку) и в логи прокси.
func (c *Client) WallGet(ctx context.Context) ([]Post, error) {
	form := url.Values{}
	if _, err := strconv.Atoi(c.group); err == nil {
		form.Set("owner_id", c.group) // числовой id, у сообществ он со знаком минус
	} else {
		form.Set("domain", c.group) // короткое имя, например "polytech"
	}
	form.Set("count", strconv.Itoa(c.count))
	form.Set("filter", "owner") // только посты самого сообщества
	form.Set("access_token", c.token)
	form.Set("v", c.version)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.BaseURL+"/wall.get", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vk request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vk http status %d", resp.StatusCode)
	}

	var body apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("vk decode: %w", err)
	}
	// Важно: VK отдаёт ошибки со статусом 200, поэтому проверяем тело.
	if body.Error != nil {
		return nil, body.Error
	}
	if body.Response == nil {
		return nil, errors.New("vk: empty response")
	}
	return body.Response.Items, nil
}
