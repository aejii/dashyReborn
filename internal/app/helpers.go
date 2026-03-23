package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"sort"
	"strings"
)

func fallback(value, fallbackValue string) string {
	if strings.TrimSpace(value) == "" {
		return fallbackValue
	}
	return value
}

func uniqueStrings(in []string) []string {
	set := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" {
			continue
		}
		if _, ok := set[v]; ok {
			continue
		}
		set[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func slugify(in string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(in)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func uniqueSlug(base string, used map[string]struct{}) string {
	if base == "" {
		base = "page"
	}
	if _, ok := used[base]; !ok {
		used[base] = struct{}{}
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := used[candidate]; !ok {
			used[candidate] = struct{}{}
			return candidate
		}
	}
}

func markerFor(primary, secondary string) string {
	source := strings.TrimSpace(primary)
	if source == "" {
		source = strings.TrimSpace(secondary)
	}
	if source == "" {
		return "•"
	}
	for _, prefix := range []string{"fas fa-", "far fa-", "fab fa-", "mdi-"} {
		source = strings.TrimPrefix(source, prefix)
	}
	source = strings.ReplaceAll(source, "-", " ")
	parts := strings.Fields(source)
	if len(parts) == 0 {
		return "•"
	}
	if len(parts) == 1 {
		r := []rune(parts[0])
		if len(r) == 0 {
			return "•"
		}
		return strings.ToUpper(string(r[0]))
	}
	a, b := []rune(parts[0]), []rune(parts[1])
	if len(a) == 0 || len(b) == 0 {
		return "•"
	}
	return strings.ToUpper(string(a[0]) + string(b[0]))
}

func sectionGrid(layout string, colCount int) string {
	switch strings.ToLower(strings.TrimSpace(layout)) {
	case "vertical", "single-column", "singlecolumn":
		return "1fr"
	}
	if colCount > 0 {
		if colCount > 6 {
			colCount = 6
		}
		return fmt.Sprintf("repeat(%d,minmax(300px,1fr))", colCount)
	}
	return "repeat(auto-fit,minmax(320px,1fr))"
}

func normalizeLanguage(raw string) string {
	language := strings.ToLower(strings.TrimSpace(raw))
	if language == "" {
		return "en"
	}
	return language
}

func normalizeItemSize(size string) string {
	switch strings.ToLower(strings.TrimSpace(size)) {
	case "small":
		return "small"
	case "large":
		return "large"
	default:
		return "medium"
	}
}

func snapshotFiles(paths []string) map[string]fileState {
	out := make(map[string]fileState, len(paths))
	for _, path := range uniqueStrings(paths) {
		info, err := os.Stat(path)
		if err != nil {
			out[path] = fileState{}
			continue
		}
		out[path] = fileState{
			Exists:  true,
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano(),
		}
	}
	return out
}

func fileStatesEqual(a, b map[string]fileState) bool {
	if len(a) != len(b) {
		return false
	}
	for path, state := range a {
		if other, ok := b[path]; !ok || other != state {
			return false
		}
	}
	return true
}

func asciiHash(input string) string {
	if strings.TrimSpace(input) == "" {
		input = "dashyreborn"
	}
	sum := 0
	for _, r := range input {
		sum += int(r)
	}
	asciiSum := fmt.Sprintf("%d", sum)
	shortened := asciiSum
	if len(shortened) > 60 {
		shortened = asciiSum[:30] + asciiSum[len(asciiSum)-30:]
	}
	return base64.StdEncoding.EncodeToString([]byte(shortened))
}
