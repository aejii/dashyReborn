package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPageSafeByDefault(t *testing.T) {
	assets := newAssetRegistry("", "", assetModeAuto)
	page := buildPage(
		"Demo",
		"",
		dashboardConfig{},
		pageInfo{
			Title:      "Demo",
			FooterText: "<script>alert(1)</script>",
		},
		appConfig{
			CustomCSS: "body{display:none}",
			Language:  "fr",
		},
		"",
		assets,
		loadOptions{AssetMode: assetModeAuto},
	)

	if page.AllowUnsafeHTML {
		t.Fatalf("footer should be escaped by default")
	}
	if page.AllowCustomCSS {
		t.Fatalf("custom CSS should be disabled by default")
	}
	if page.Language != "fr" {
		t.Fatalf("language = %q, want fr", page.Language)
	}

	tmpl, err := newTemplate()
	if err != nil {
		t.Fatalf("newTemplate: %v", err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, pageContext{Page: page}); err != nil {
		t.Fatalf("execute template: %v", err)
	}

	rendered := out.String()
	if strings.Contains(rendered, "<script>alert(1)</script>") {
		t.Fatalf("footer HTML should be escaped in rendered output")
	}
	if !strings.Contains(rendered, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Fatalf("escaped footer not found: %s", rendered)
	}
	if strings.Contains(rendered, "body{display:none}") {
		t.Fatalf("custom CSS should not be rendered by default")
	}
}

func TestBuildPageUnsafeModesAreExplicitlyOptIn(t *testing.T) {
	assets := newAssetRegistry("", "", assetModeAuto)
	page := buildPage(
		"Demo",
		"",
		dashboardConfig{},
		pageInfo{
			Title:      "Demo",
			FooterText: "<strong>Footer</strong>",
		},
		appConfig{
			CustomCSS: "body{display:none}",
		},
		"",
		assets,
		loadOptions{
			AssetMode:       assetModeAuto,
			AllowUnsafeHTML: true,
			AllowUnsafeCSS:  true,
		},
	)

	tmpl, err := newTemplate()
	if err != nil {
		t.Fatalf("newTemplate: %v", err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, pageContext{Page: page}); err != nil {
		t.Fatalf("execute template: %v", err)
	}

	rendered := out.String()
	if !strings.Contains(rendered, "<strong>Footer</strong>") {
		t.Fatalf("unsafe footer should be rendered when opt-in is enabled")
	}
	if !strings.Contains(rendered, "body{display:none}") {
		t.Fatalf("unsafe custom CSS should be rendered when opt-in is enabled")
	}
}

func TestResolveIconOfflineFallback(t *testing.T) {
	assets := newAssetRegistry("", "", assetModeOffline)
	iconURL, iconClass, fallback, loadFA, loadMDI, _ := assets.resolveIcon("si-grafana", "https://grafana.example.com", "")
	if iconURL != "" || iconClass != "" || loadFA || loadMDI {
		t.Fatalf("expected no external asset in offline mode, got url=%q class=%q", iconURL, iconClass)
	}
	if strings.TrimSpace(fallback) == "" {
		t.Fatalf("expected fallback marker in offline mode")
	}
}

func TestLoadSiteResolvesRelativeAssetsAcrossPages(t *testing.T) {
	dir := t.TempDir()
	writeTempFileInDir(t, dir, "logo.png", "logo")
	writeTempFileInDir(t, dir, "page-logo.png", "page-logo")
	writeTempConfigInDir(t, dir, "page.yml", `
pageInfo:
  title: Child
  logo: ./page-logo.png
sections:
  - name: Child
    items:
      - title: Grafana
        url: http://example.com
`)
	root := writeTempConfigInDir(t, dir, "config.yml", `
pageInfo:
  title: Root
  logo: ./logo.png
pages:
  - name: Child
    path: ./page.yml
sections:
  - name: Root
    items:
      - title: Home
        url: http://example.com
`)

	site, err := loadSite(root, "", loadOptions{AssetMode: assetModeAuto})
	if err != nil {
		t.Fatalf("loadSite: %v", err)
	}
	if len(site.Pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(site.Pages))
	}
	if !strings.HasPrefix(site.Pages[0].Logo, "/_local-assets/") {
		t.Fatalf("expected root logo to be served locally, got %q", site.Pages[0].Logo)
	}
	if !strings.HasPrefix(site.Pages[1].Logo, "/_local-assets/") {
		t.Fatalf("expected child logo to be served locally, got %q", site.Pages[1].Logo)
	}
	if len(site.LocalAssets) != 2 {
		t.Fatalf("expected 2 registered local assets, got %d", len(site.LocalAssets))
	}
}

func TestAssetRegistryServesFilesInsidePublicDirWithAssetsPrefix(t *testing.T) {
	publicDir := filepath.Join(t.TempDir(), "public")
	if err := os.MkdirAll(filepath.Join(publicDir, "images"), 0o755); err != nil {
		t.Fatalf("mkdir public dir: %v", err)
	}

	assets := newAssetRegistry(publicDir, "", assetModeAuto)
	got := assets.resolve(filepath.Join(publicDir, "images", "logo.png"), "")
	if got != "/_assets/images/logo.png" {
		t.Fatalf("assets.resolve() = %q, want %q", got, "/_assets/images/logo.png")
	}
}

func TestBuildPageUsesEnglishUIStringsByDefault(t *testing.T) {
	assets := newAssetRegistry("", "", assetModeAuto)
	page := buildPage(
		"Demo",
		"",
		dashboardConfig{},
		pageInfo{Title: "Demo"},
		appConfig{},
		"",
		assets,
		loadOptions{AssetMode: assetModeAuto},
	)

	if page.UI.SearchLabel != "Search" {
		t.Fatalf("SearchLabel = %q, want %q", page.UI.SearchLabel, "Search")
	}
	if page.UI.NoResults != "No results." {
		t.Fatalf("NoResults = %q, want %q", page.UI.NoResults, "No results.")
	}
}

func TestCardStyleRejectsUnsafeValues(t *testing.T) {
	style := cardStyle(`red;display:none`, `url(https://example.com/x)`)
	if style != "" {
		t.Fatalf("cardStyle() = %q, want empty string", style)
	}

	style = cardStyle("#fff", "rgba(0, 0, 0, 0.5)")
	if !strings.Contains(style, "--item-text-color:#fff") {
		t.Fatalf("expected safe color to be preserved, got %q", style)
	}
}
