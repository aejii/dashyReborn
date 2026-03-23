package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T, yaml string, opts loadOptions) *appServer {
	t.Helper()

	path := writeTempConfig(t, yaml)
	srv, err := newServer(path, "", opts)
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	return srv
}

func TestServeHealthReturnsWarningsAndStatus(t *testing.T) {
	srv := newTestServer(t, `
appConfig:
  statusCheck: true
sections:
  - name: Services
    items:
      - title: Grafana
        url: http://example.com
`, loadOptions{AssetMode: assetModeAuto})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.serveHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", rec.Code)
	}

	var payload healthPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode health payload: %v", err)
	}
	if payload.Status != "ok" {
		t.Fatalf("status = %q, want ok", payload.Status)
	}
	if !containsWarning(payload.Warnings, "ignored field appConfig.statusCheck") {
		t.Fatalf("missing warning in health payload: %v", payload.Warnings)
	}
	if payload.Version != appVersion {
		t.Fatalf("version = %q, want %q", payload.Version, appVersion)
	}
	if payload.UnsafeHTML || payload.UnsafeCSS {
		t.Fatalf("unsafe flags should be false by default: %#v", payload)
	}
	if payload.ReloadSuccessCount != 1 || payload.ReloadFailureCount != 0 {
		t.Fatalf("unexpected reload counters: %#v", payload)
	}
}

func TestServePageRendersConfiguredLanguage(t *testing.T) {
	srv := newTestServer(t, `
appConfig:
  language: fr
sections:
  - name: Services
    items:
      - title: Grafana
        url: http://example.com
`, loadOptions{AssetMode: assetModeAuto})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.servePage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("page status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `<html lang="fr">`) {
		t.Fatalf("expected rendered page language to be fr")
	}
}

func TestServeLocalAssetReturnsRegisteredFile(t *testing.T) {
	dir := t.TempDir()
	writeTempFileInDir(t, dir, "logo.png", "logo")
	configPath := writeTempConfigInDir(t, dir, "config.yml", `
pageInfo:
  title: Demo
  logo: ./logo.png
sections:
  - name: Services
    items:
      - title: Grafana
        url: http://example.com
`)

	srv, err := newServer(configPath, "", loadOptions{AssetMode: assetModeAuto})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	site := srv.currentSite()
	if site == nil || len(site.LocalAssets) != 1 {
		t.Fatalf("expected one local asset, got %#v", site)
	}
	var route string
	for id := range site.LocalAssets {
		route = "/_local-assets/" + id
	}

	req := httptest.NewRequest(http.MethodGet, route, nil)
	rec := httptest.NewRecorder()
	srv.serveLocalAsset(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("serveLocalAsset status = %d, want 200", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "logo" {
		t.Fatalf("serveLocalAsset body = %q, want %q", rec.Body.String(), "logo")
	}
}

func TestServeLocalAssetRejectsUnknownIDs(t *testing.T) {
	srv := newTestServer(t, `
pageInfo:
  title: Demo
sections:
  - name: Services
    items:
      - title: Grafana
        url: http://example.com
`, loadOptions{AssetMode: assetModeAuto})

	req := httptest.NewRequest(http.MethodGet, "/_local-assets/unknown", nil)
	rec := httptest.NewRecorder()
	srv.serveLocalAsset(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("serveLocalAsset status = %d, want 404", rec.Code)
	}
}

func TestServePageReturnsNotFoundForUnknownSlug(t *testing.T) {
	srv := newTestServer(t, `
pageInfo:
  title: Demo
sections:
  - name: Services
    items:
      - title: Grafana
        url: http://example.com
`, loadOptions{AssetMode: assetModeAuto})

	req := httptest.NewRequest(http.MethodGet, "/page/unknown", nil)
	rec := httptest.NewRecorder()
	srv.servePage(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("servePage status = %d, want 404", rec.Code)
	}
}

func TestHealthStatusBecomesDegradedAfterReloadFailure(t *testing.T) {
	path := writeTempConfig(t, `
sections:
  - name: Services
    items:
      - title: Grafana
        url: http://example.com
`)

	srv, err := newServer(path, "", loadOptions{AssetMode: assetModeAuto})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	if err := os.WriteFile(path, []byte("pageInfo: ["), 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}
	if err := srv.reload(); err == nil {
		t.Fatalf("expected reload to fail")
	}
	status := srv.healthStatus()
	if status.Status != "degraded" {
		t.Fatalf("health status = %q, want degraded", status.Status)
	}
	if status.LastReloadError == "" {
		t.Fatalf("expected last reload error to be populated")
	}
	if status.ReloadFailureCount != 1 {
		t.Fatalf("reload failure count = %d, want 1", status.ReloadFailureCount)
	}
}

func TestServeEventsTracksActiveClients(t *testing.T) {
	srv := newTestServer(t, `
pageInfo:
  title: Demo
sections:
  - name: Services
    items:
      - title: Grafana
        url: http://example.com
`, loadOptions{AssetMode: assetModeAuto})

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		srv.serveEvents(rec, req)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if srv.hub.count() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := srv.healthStatus().ActiveSSEClients; got != 1 {
		cancel()
		t.Fatalf("ActiveSSEClients = %d, want 1", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("serveEvents did not exit after context cancellation")
	}
	if got := srv.healthStatus().ActiveSSEClients; got != 0 {
		t.Fatalf("ActiveSSEClients = %d, want 0 after disconnect", got)
	}
}

func TestWatchReloadsLocalConfigChanges(t *testing.T) {
	path := writeTempConfig(t, `
pageInfo:
  title: Before
sections:
  - name: Services
    items:
      - title: Grafana
        url: http://example.com
`)

	srv, err := newServer(path, "", loadOptions{AssetMode: assetModeAuto})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	go srv.watch(20 * time.Millisecond)

	time.Sleep(40 * time.Millisecond)
	if err := os.WriteFile(path, []byte(`
pageInfo:
  title: After
sections:
  - name: Services
    items:
      - title: Grafana
        url: http://example.com
`), 0o600); err != nil {
		t.Fatalf("write updated config: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		site := srv.currentSite()
		if site != nil && len(site.Pages) > 0 && site.Pages[0].Title == "After" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("watch did not reload updated config in time")
}

func TestServeFaviconCacheFetchesAndPersistsIcon(t *testing.T) {
	iconBytes := []byte{0x89, 0x50, 0x4e, 0x47}
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<html><head><link rel="icon" href="/favicon.png"></head></html>`))
		case "/favicon.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(iconBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer remote.Close()

	cacheDir := t.TempDir()
	srv := newTestServer(t, `
sections:
  - name: Services
    items:
      - title: Remote
        url: `+remote.URL+`
        icon: favicon
`, loadOptions{AssetMode: assetModeInternalOnly, FaviconCacheDir: cacheDir})

	site := srv.currentSite()
	if site == nil || len(site.FaviconTargets) != 1 {
		t.Fatalf("expected one favicon target, got %#v", site)
	}
	var route string
	for id := range site.FaviconTargets {
		route = "/_favicon-cache/" + id
	}

	req := httptest.NewRequest(http.MethodGet, route, nil)
	rec := httptest.NewRecorder()
	srv.serveFaviconCache(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("serveFaviconCache status = %d, want 200", rec.Code)
	}
	if !bytes.Equal(rec.Body.Bytes(), iconBytes) {
		t.Fatalf("unexpected favicon body: %v", rec.Body.Bytes())
	}
	if files, _ := os.ReadDir(cacheDir); len(files) != 1 {
		t.Fatalf("expected one cached favicon file, got %d", len(files))
	}
}

func TestServeFaviconCacheOfflineRequiresExistingCache(t *testing.T) {
	cacheDir := t.TempDir()
	srv := newTestServer(t, `
sections:
  - name: Services
    items:
      - title: Remote
        url: https://example.com
        icon: favicon
`, loadOptions{AssetMode: assetModeOffline, FaviconCacheDir: cacheDir})

	site := srv.currentSite()
	var route string
	for id := range site.FaviconTargets {
		route = "/_favicon-cache/" + id
	}

	req := httptest.NewRequest(http.MethodGet, route, nil)
	rec := httptest.NewRecorder()
	srv.serveFaviconCache(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("serveFaviconCache status = %d, want 404", rec.Code)
	}
}
