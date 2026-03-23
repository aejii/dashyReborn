package app

import (
	"crypto/sha1"
	"encoding/hex"
	"path/filepath"
	"strings"
)

type assetRegistry struct {
	publicDir       string
	faviconCacheDir string
	mode            assetMode
	faviconTargets  map[string]string
	localFiles      map[string]string
	remoteRefs      map[string]struct{}
}

const (
	fontAwesomeCSSURL  = "https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.2/css/all.min.css"
	mdiCSSURL          = "https://cdn.jsdelivr.net/npm/@mdi/font@7.0.96/css/materialdesignicons.min.css"
	simpleIconsBase    = "https://unpkg.com/simple-icons@v7/icons/"
	dashboardIconsBase = "https://raw.githubusercontent.com/walkxcode/dashboard-icons/master/png/"
	selfhIconsBase     = "https://cdn.jsdelivr.net/gh/selfhst/icons@latest/webp/"
	dicebearURL        = "https://api.dicebear.com/7.x/identicon/svg?seed="
)

func newAssetRegistry(publicDir, faviconCacheDir string, mode assetMode) *assetRegistry {
	return &assetRegistry{
		publicDir:       publicDir,
		faviconCacheDir: strings.TrimSpace(faviconCacheDir),
		mode:            mode,
		faviconTargets:  make(map[string]string),
		localFiles:      make(map[string]string),
		remoteRefs:      make(map[string]struct{}),
	}
}

func (r *assetRegistry) snapshot() map[string]string {
	out := make(map[string]string, len(r.localFiles))
	for key, value := range r.localFiles {
		out[key] = value
	}
	return out
}

func (r *assetRegistry) remoteAssets() []string {
	out := make([]string, 0, len(r.remoteRefs))
	for ref := range r.remoteRefs {
		out = append(out, ref)
	}
	return uniqueStrings(out)
}

func (r *assetRegistry) faviconEntries() map[string]string {
	out := make(map[string]string, len(r.faviconTargets))
	for key, value := range r.faviconTargets {
		out[key] = value
	}
	return out
}

func (r *assetRegistry) resolve(raw, baseDir string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if isRemote(raw) {
		if !r.mode.allowsRemoteAssets() {
			return ""
		}
		r.noteRemote(raw)
		return raw
	}
	if isRemote(baseDir) {
		resolved := resolveConfigReference(baseDir, raw)
		if isRemote(resolved) {
			if !r.mode.allowsRemoteAssets() {
				return ""
			}
			r.noteRemote(resolved)
			return resolved
		}
	}
	if strings.HasPrefix(raw, "/") && r.publicDir != "" {
		return "/_assets" + raw
	}

	resolved := resolveConfigReference(baseDir, raw)
	if isRemote(resolved) {
		if !r.mode.allowsRemoteAssets() {
			return ""
		}
		r.noteRemote(resolved)
		return resolved
	}
	if filepath.IsAbs(resolved) {
		return r.registerLocalFile(resolved)
	}
	if baseDir != "" && !isRemote(baseDir) {
		return r.registerLocalFile(filepath.Join(baseDir, raw))
	}
	return raw
}

func (r *assetRegistry) resolveIcon(icon, itemURL, baseDir string) (string, string, string, bool, bool, bool) {
	icon = strings.TrimSpace(icon)
	fallback := markerFor(icon, itemURL)
	switch {
	case icon == "":
		return "", "", markerFor(itemURL, ""), false, false, false
	case icon == "generative":
		if !r.mode.allowsRemoteAssets() {
			return "", "", fallback, false, false, false
		}
		url := generativeIconURL(itemURL)
		r.noteRemote(url)
		return url, "", fallback, false, false, false
	case strings.EqualFold(icon, "favicon") && itemURL != "":
		return r.registerFavicon(itemURL), "", fallback, false, false, true
	case isRemote(icon):
		if !r.mode.allowsRemoteAssets() {
			return "", "", fallback, false, false, false
		}
		r.noteRemote(icon)
		return icon, "", fallback, false, false, false
	case strings.Contains(icon, "fa-"):
		if !r.mode.allowsRemoteAssets() {
			return "", "", fallback, false, false, false
		}
		return "", normalizeFontAwesomeClass(icon), fallback, true, false, false
	case strings.HasPrefix(icon, "mdi-"):
		if !r.mode.allowsRemoteAssets() {
			return "", "", fallback, false, false, false
		}
		return "", "mdi " + icon, fallback, false, true, false
	case strings.HasPrefix(icon, "si-"):
		if !r.mode.allowsRemoteAssets() {
			return "", "", fallback, false, false, false
		}
		url := simpleIconsBase + strings.ToLower(strings.TrimPrefix(icon, "si-")) + ".svg"
		r.noteRemote(url)
		return url, "", fallback, false, false, false
	case strings.HasPrefix(icon, "hl-"):
		if !r.mode.allowsRemoteAssets() {
			return "", "", fallback, false, false, false
		}
		url := dashboardIconsBase + strings.ToLower(strings.TrimPrefix(icon, "hl-")) + ".png"
		r.noteRemote(url)
		return url, "", fallback, false, false, false
	case strings.HasPrefix(icon, "sh-"):
		if !r.mode.allowsRemoteAssets() {
			return "", "", fallback, false, false, false
		}
		url := selfhIconsBase + strings.ToLower(strings.TrimPrefix(icon, "sh-")) + ".webp"
		r.noteRemote(url)
		return url, "", fallback, false, false, false
	case looksLikeAssetReference(icon):
		return r.resolve(icon, baseDir), "", fallback, false, false, false
	default:
		return "", "", fallback, false, false, false
	}
}

func (r *assetRegistry) registerLocalFile(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	if rel, ok := pathWithin(r.publicDir, abs); ok {
		return "/_assets/" + filepath.ToSlash(rel)
	}
	id := stableAssetID(abs)
	r.localFiles[id] = abs
	return "/_local-assets/" + id
}

func looksLikeAssetReference(raw string) bool {
	if raw == "" {
		return false
	}
	if filepath.IsAbs(raw) || strings.HasPrefix(raw, ".") || strings.ContainsAny(raw, `/\`) {
		return true
	}
	switch strings.ToLower(filepath.Ext(raw)) {
	case ".png", ".svg", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".bmp", ".avif":
		return true
	default:
		return false
	}
}

func pathWithin(root, target string) (string, bool) {
	if root == "" {
		return "", false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(absRoot, target)
	if err != nil || rel == "." || rel == "" || filepath.IsAbs(rel) {
		return "", false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}

func stableAssetID(input string) string {
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func (r *assetRegistry) registerFavicon(targetURL string) string {
	id := stableAssetID(strings.TrimSpace(targetURL))
	r.faviconTargets[id] = strings.TrimSpace(targetURL)
	return "/_favicon-cache/" + id
}

func (r *assetRegistry) noteRemote(ref string) {
	if strings.TrimSpace(ref) == "" {
		return
	}
	r.remoteRefs[ref] = struct{}{}
}

func generativeIconURL(raw string) string {
	return dicebearURL + queryEscape(asciiHash(raw))
}
