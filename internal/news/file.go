package news

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type snapshot struct {
	UpdatedAt time.Time `json:"updated_at"`
	Items     []News    `json:"items"`
}

func LoadFile(
	path string,
) (
	[]News,
	time.Time,
	error,
) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	var s snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, time.Time{}, fmt.Errorf("parse %s: %w", path, err)
	}

	for i := range s.Items {
		if id, ok := SourceIDFromSlug(s.Items[i].ID); ok {
			s.Items[i].SourceID = id
		}
	}

	// В файлах старого формата нет updated_at — берём время изменения файла.
	if s.UpdatedAt.IsZero() {
		if fi, err := os.Stat(path); err == nil {
			s.UpdatedAt = fi.ModTime().UTC()
		}
	}
	return s.Items, s.UpdatedAt, nil
}

// SaveFile атомарно перезаписывает файл: пишет во временный файл рядом
// и переименовывает его. Читатель никогда не увидит полузаписанный JSON,
// а при ошибке старый файл остаётся нетронутым.
func SaveFile(
	path string,
	items []News,
	updatedAt time.Time,
) error {
	if items == nil {
		items = []News{}
	}
	data, err := json.MarshalIndent(snapshot{UpdatedAt: updatedAt, Items: items}, "", "    ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // после успешного Rename файла уже нет — ошибка игнорируется

	_, err = tmp.Write(append(data, '\n'))
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), 0o644) // CreateTemp создаёт файл с правами 0600
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		return fmt.Errorf("save %s: %w", path, err)
	}
	return nil
}

// IsNotExist сообщает, что файла с новостями ещё нет (первый запуск).
func IsNotExist(
	err error,
) bool {
	return errors.Is(err, os.ErrNotExist)
}
