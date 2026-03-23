package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	linkTagPattern  = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	attrPattern     = regexp.MustCompile(`(?is)([a-zA-Z:-]+)\s*=\s*("([^"]*)"|'([^']*)'|([^\s>]+))`)
	faviconExts     = []string{".ico", ".png", ".svg", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".avif"}
	errFaviconFetch = errors.New("favicon fetch disabled")
)

func findCachedFavicon(cacheDir, id string) (string, bool) {
	if strings.TrimSpace(cacheDir) == "" || strings.TrimSpace(id) == "" {
		return "", false
	}
	for _, ext := range faviconExts {
		path := filepath.Join(cacheDir, id+ext)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

func fetchAndCacheFavicon(cacheDir, targetURL string) (string, error) {
	if strings.TrimSpace(cacheDir) == "" {
		return "", fmt.Errorf("favicon cache dir is empty")
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}

	id := stableAssetID(strings.TrimSpace(targetURL))
	if path, ok := findCachedFavicon(cacheDir, id); ok {
		return path, nil
	}

	iconURL, err := discoverFaviconURL(targetURL)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(iconURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("favicon returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	ext := faviconExtension(resp.Header.Get("Content-Type"), iconURL, body)
	path := filepath.Join(cacheDir, id+ext)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func discoverFaviconURL(pageURL string) (string, error) {
	parsed, err := neturl.Parse(strings.TrimSpace(pageURL))
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(parsed.String())
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode < 300 && strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/html") {
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
			if readErr == nil {
				if href, ok := extractFaviconHref(parsed, body); ok {
					return href, nil
				}
			}
		}
	}

	candidates := []string{"/favicon.ico", "/favicon.png", "/apple-touch-icon.png", "/favicon.svg"}
	for _, candidate := range candidates {
		if href, ok := resolveAndProbeFavicon(client, parsed, candidate); ok {
			return href, nil
		}
	}
	return "", fmt.Errorf("unable to discover favicon for %s", pageURL)
}

func extractFaviconHref(base *neturl.URL, body []byte) (string, bool) {
	tags := linkTagPattern.FindAll(body, -1)
	if len(tags) == 0 {
		return "", false
	}
	type candidate struct {
		score int
		href  string
	}
	best := candidate{}
	for _, tag := range tags {
		attrs := parseHTMLAttrs(string(tag))
		rel := strings.ToLower(strings.TrimSpace(attrs["rel"]))
		href := strings.TrimSpace(attrs["href"])
		if href == "" || rel == "" {
			continue
		}
		score := faviconRelScore(rel)
		if score == 0 {
			continue
		}
		resolved, err := base.Parse(href)
		if err != nil {
			continue
		}
		if score > best.score {
			best = candidate{score: score, href: resolved.String()}
		}
	}
	return best.href, best.href != ""
}

func parseHTMLAttrs(tag string) map[string]string {
	out := make(map[string]string)
	matches := attrPattern.FindAllStringSubmatch(tag, -1)
	for _, match := range matches {
		key := strings.ToLower(strings.TrimSpace(match[1]))
		value := match[3]
		if value == "" {
			value = match[4]
		}
		if value == "" {
			value = match[5]
		}
		out[key] = strings.TrimSpace(value)
	}
	return out
}

func faviconRelScore(rel string) int {
	switch {
	case strings.Contains(rel, "apple-touch-icon"):
		return 4
	case strings.Contains(rel, "shortcut icon"):
		return 3
	case strings.Contains(rel, "icon"):
		return 2
	case strings.Contains(rel, "mask-icon"):
		return 1
	default:
		return 0
	}
}

func resolveAndProbeFavicon(client *http.Client, base *neturl.URL, candidate string) (string, bool) {
	resolved, err := base.Parse(candidate)
	if err != nil {
		return "", false
	}
	req, err := http.NewRequest(http.MethodHead, resolved.String(), nil)
	if err != nil {
		return "", false
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", false
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", false
	}
	return resolved.String(), true
}

func faviconExtension(contentType, sourceURL string, body []byte) string {
	lowered := strings.ToLower(contentType)
	switch {
	case strings.Contains(lowered, "image/svg"):
		return ".svg"
	case strings.Contains(lowered, "image/png"):
		return ".png"
	case strings.Contains(lowered, "image/webp"):
		return ".webp"
	case strings.Contains(lowered, "image/jpeg"):
		return ".jpg"
	case strings.Contains(lowered, "image/gif"):
		return ".gif"
	case strings.Contains(lowered, "image/bmp"):
		return ".bmp"
	case strings.Contains(lowered, "image/avif"):
		return ".avif"
	case strings.Contains(lowered, "image/x-icon"), strings.Contains(lowered, "image/vnd.microsoft.icon"):
		return ".ico"
	}
	if ext := strings.ToLower(filepath.Ext(sourceURL)); ext != "" {
		for _, allowed := range faviconExts {
			if ext == allowed {
				return ext
			}
		}
	}
	if bytes.HasPrefix(body, []byte("<svg")) || bytes.Contains(body[:min(128, len(body))], []byte("<svg")) {
		return ".svg"
	}
	return ".ico"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
