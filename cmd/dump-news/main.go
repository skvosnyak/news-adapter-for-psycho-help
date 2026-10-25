// dump-news выкачивает все посты со стены VK и сохраняет их в JSON-файл
// в формате ответа /api/news ({"items": [...]}).
//
//	go run ./cmd/dump-news               # → store/news.json
//	go run ./cmd/dump-news -o other.json
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"news-service/internal/config"
	"news-service/internal/news"
	"news-service/internal/vk"
)

func main() {
	out := flag.String("o", "store/news.json", "куда записать новости")
	flag.Parse()

	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(out string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	client := vk.NewClient(cfg.VKToken, cfg.VKAPIVersion, cfg.VKGroup, cfg.VKPostsCount)
	posts, err := client.WallGetAll(ctx)
	if err != nil {
		return err
	}
	items := vk.ToNews(posts)

	data, err := json.MarshalIndent(struct {
		Items []news.News `json:"items"`
	}{items}, "", "    ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	// Пишем во временный файл и переименовываем, чтобы не оставить
	// полузаписанный JSON, если что-то пойдёт не так.
	tmp := out + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, out); err != nil {
		return err
	}

	fmt.Printf("posts from VK: %d, news written: %d → %s\n", len(posts), len(items), out)
	return nil
}
