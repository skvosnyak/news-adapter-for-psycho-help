// Package config читает настройки сервиса из переменных окружения.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port            string        // порт HTTP-сервера
	VKToken         string        // сервисный ключ приложения VK
	VKGroup         string        // короткое имя паблика (domain) или его id со знаком минус
	VKAPIVersion    string        // версия VK API
	VKPostsCount    int           // сколько последних постов забирать (максимум 100)
	RefreshInterval time.Duration // как часто обновлять новости
	RetryInterval   time.Duration // через сколько повторить, если VK не ответил
	CORSOrigin      string        // адрес фронта, если он на другом домене/порту (иначе пусто)
	LogFile         string        // файл, куда дублировать логи (иначе только stdout)
}

func Load() (Config, error) {
	loadDotEnv(".env")

	cfg := Config{
		Port:         getEnv("PORT", "8080"),
		VKToken:      os.Getenv("VK_SERVICE_TOKEN"),
		VKGroup:      getEnv("VK_GROUP", "spp_polytech"),
		VKAPIVersion: getEnv("VK_API_VERSION", "5.199"),
		CORSOrigin:   os.Getenv("CORS_ORIGIN"),
		LogFile:      os.Getenv("LOG_FILE"),
	}

	var err error
	if cfg.VKPostsCount, err = getInt("VK_POSTS_COUNT", 100); err != nil {
		return Config{}, err
	}
	if cfg.RefreshInterval, err = getDuration("REFRESH_INTERVAL", time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.RetryInterval, err = getDuration("RETRY_INTERVAL", 5*time.Minute); err != nil {
		return Config{}, err
	}

	if cfg.VKToken == "" {
		return Config{}, errors.New("VK_SERVICE_TOKEN is required")
	}
	if cfg.VKPostsCount < 1 || cfg.VKPostsCount > 100 {
		return Config{}, errors.New("VK_POSTS_COUNT must be between 1 and 100")
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return n, nil
}

// getDuration читает длительность вида "1h", "3h", "30m".
func getDuration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}
	return d, nil
}

// loadDotEnv — минимальный загрузчик .env без сторонних библиотек.
// Формат: KEY=VALUE
// Уже заданные переменные окружения НЕ перезаписываются.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		raw := val
		val = strings.TrimSpace(val)

		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
			val = val[1 : len(val)-1]
		} else {
			val, _, _ = strings.Cut(raw, " #")
			val = strings.TrimSpace(val)
		}

		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: reading %s: %v\n", path, err)
	}
}
