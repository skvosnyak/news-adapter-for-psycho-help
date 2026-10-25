package news

import (
	"strconv"
	"strings"
)

const maxSlugBaseRunes = 60

var translit = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// Slugify делает из заголовка и номера поста человекочитаемый id для URL:
// Номер в конце гарантирует уникальность (заголовки могут совпадать)
func Slugify(title string, sourceID int) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		if t, ok := translit[r]; ok {
			b.WriteString(t)
			dash = false
		} else if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-') // пробелы, знаки, эмодзи → один дефис
			dash = true
		}
	}
	base := strings.Trim(b.String(), "-")

	// Ограничиваем длину, обрезая по границе слова.
	if len(base) > maxSlugBaseRunes { // после транслита тут только ASCII
		base = base[:maxSlugBaseRunes]
		if i := strings.LastIndexByte(base, '-'); i > 0 {
			base = base[:i]
		}
	}

	id := strconv.Itoa(sourceID)
	if base == "" {
		return id
	}
	return base + "-" + id
}

// SourceIDFromSlug достаёт номер поста из конца slug'а ("...-512" → 512).
func SourceIDFromSlug(slug string) (int, bool) {
	tail := slug
	if i := strings.LastIndexByte(slug, '-'); i >= 0 {
		tail = slug[i+1:]
	}
	id, err := strconv.Atoi(tail)
	return id, err == nil
}
