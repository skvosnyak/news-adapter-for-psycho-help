package vk

import (
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"news-service/internal/news"
)

const maxTitleRunes = 100

// mentionRe находит упоминания вида [club123|Название] или [id1|Имя].
var mentionRe = regexp.MustCompile(`\[(?:id|club|public)\d+\|([^\]]+)\]`)

// ToNews — сердце адаптера: переводит посты VK в наш формат.
// Отбрасывает рекламу, репосты и посты без текста (без текста нечего ставить
// в заголовок), сортирует от новых к старым — закреплённый пост VK ставит
// первым, даже если он старый.
func ToNews(posts []Post) []news.News {
	result := make([]news.News, 0, len(posts))
	for _, p := range posts {
		if p.MarkedAsAds == 1 || len(p.CopyHistory) > 0 {
			continue
		}
		n, ok := convertPost(p)
		if !ok {
			continue
		}
		result = append(result, n)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Date.After(result[j].Date)
	})
	return result
}

// convertPost переводит один пост. ok=false — пост без текста, пропускаем.
func convertPost(p Post) (news.News, bool) {
	text := strings.TrimSpace(mentionRe.ReplaceAllString(p.Text, "$1"))
	if text == "" {
		return news.News{}, false
	}
	title, body := splitTitle(text)

	return news.News{
		ID:       news.Slugify(title, p.ID),
		Images:   photoURLs(p.Attachments),
		Type:     nil, // в VK нет типов новостей
		Date:     time.Unix(p.Date, 0).UTC(),
		Title:    title,
		Text:     body,
		SourceID: p.ID,
	}, true
}

// splitTitle придумывает заголовок, которого у постов VK нет.
//   - Первая строка короткая (≤100 символов) — обычно это и есть заголовок:
//     title = первая строка, text = остальное.
//   - Первая строка длинная — это абзац: title = первые ~100 символов + "...",
//     text = весь текст целиком (чтобы ничего не потерялось).
//
// text = nil, если после заголовка ничего не осталось.
func splitTitle(text string) (string, *string) {
	firstLine, rest, _ := strings.Cut(text, "\n")
	firstLine = strings.TrimSpace(firstLine)

	var title, body string
	if len([]rune(firstLine)) <= maxTitleRunes {
		title, body = firstLine, strings.TrimSpace(rest)
	} else {
		title, body = truncateWords(firstLine, maxTitleRunes)+"...", text
	}

	if body == "" {
		return title, nil
	}
	return title, &body
}

// truncateWords обрезает строку до n символов (именно символов-рун, а не байт:
// кириллическая буква в UTF-8 занимает 2 байта) и старается не резать слово.
func truncateWords(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	cut := r[:n]
	// Если обрезали посреди слова — откатываемся к последнему пробелу,
	// но не дальше, чем на 30 символов.
	if !unicode.IsSpace(r[n]) {
		for i := len(cut) - 1; i >= n-30 && i > 0; i-- {
			if unicode.IsSpace(cut[i]) {
				cut = cut[:i]
				break
			}
		}
	}
	return strings.TrimRight(string(cut), " ,.;:—–-")
}

// Размеры фото в VK: каждое фото приходит в ~10 копиях.
//
//	s 75px, m 130px, x 604px, y 807px, z 1080×1024, w 2560×2048 —
//	  пропорциональные копии (соотношение сторон как у оригинала);
//	o, p, q, r — обрезанные VK превью (crop), их не берём.
//
// Берём первую найденную по приоритету: z — хорошее качество
// и обычно 100–300 КБ; w — уже почти оригинал и может весить мегабайты.
var sizePriority = []string{"z", "y", "x", "w", "m", "s"}

var croppedSizes = map[string]bool{"o": true, "p": true, "q": true, "r": true}

// photoURLs возвращает по одной ссылке на каждое фото поста.
// Пустой срез, а не nil, чтобы в JSON было [], а не null.
func photoURLs(atts []Attachment) []string {
	urls := []string{}
	for _, a := range atts {
		if a.Type != "photo" || a.Photo == nil {
			continue
		}
		if u := pickPhotoSize(a.Photo.Sizes); u != "" {
			urls = append(urls, u)
		}
	}
	return urls
}

// pickPhotoSize выбирает одну копию фото: сначала по приоритету типов,
// а если VK прислал незнакомые типы — самую большую необрезанную.
func pickPhotoSize(sizes []PhotoSize) string {
	byType := make(map[string]string, len(sizes))
	for _, s := range sizes {
		byType[s.Type] = s.URL
	}
	for _, t := range sizePriority {
		if u := byType[t]; u != "" {
			return u
		}
	}

	best := ""
	bestArea := -1
	for _, s := range sizes {
		if croppedSizes[s.Type] || s.URL == "" {
			continue
		}
		if area := s.Width * s.Height; area > bestArea {
			best, bestArea = s.URL, area
		}
	}
	return best
}
