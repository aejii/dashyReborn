package app

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lrw, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, lrw.status, time.Since(start).Round(time.Millisecond))
	})
}

type eventHub struct {
	mu   sync.RWMutex
	subs map[chan string]struct{}
}

func newEventHub() *eventHub {
	return &eventHub{subs: make(map[chan string]struct{})}
}

func (h *eventHub) subscribe() chan string {
	ch := make(chan string, 1)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *eventHub) unsubscribe(ch chan string) {
	h.mu.Lock()
	delete(h.subs, ch)
	close(ch)
	h.mu.Unlock()
}

func (h *eventHub) broadcast(v string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs {
		select {
		case ch <- v:
		default:
		}
	}
}

func (h *eventHub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}

type appServer struct {
	configPath string
	publicDir  string
	options    loadOptions
	hub        *eventHub
	tmpl       *template.Template

	mu                   sync.RWMutex
	site                 *siteData
	lastReloadAttempt    time.Time
	lastSuccessfulReload time.Time
	lastReloadError      string
	reloadSuccessCount   int
	reloadFailureCount   int
}

func newServer(configPath, publicDir string, opts loadOptions) (*appServer, error) {
	tmpl, err := newTemplate()
	if err != nil {
		return nil, err
	}
	s := &appServer{
		configPath: configPath,
		publicDir:  publicDir,
		options:    opts,
		hub:        newEventHub(),
		tmpl:       tmpl,
	}
	if err := s.reload(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *appServer) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.servePage)
	mux.HandleFunc("/page/", s.servePage)
	mux.HandleFunc("/events", s.serveEvents)
	mux.HandleFunc("/healthz", s.serveHealth)
	mux.HandleFunc("/_favicon-cache/", s.serveFaviconCache)
	mux.HandleFunc("/_local-assets/", s.serveLocalAsset)
	if s.publicDir != "" {
		mux.Handle("/_assets/", http.StripPrefix("/_assets/", http.FileServer(http.Dir(s.publicDir))))
	}
}

func (s *appServer) servePage(w http.ResponseWriter, r *http.Request) {
	site := s.currentSite()
	if site == nil {
		http.Error(w, "site unavailable", http.StatusServiceUnavailable)
		return
	}

	slug := ""
	switch {
	case r.URL.Path == "/" || r.URL.Path == "":
	case r.URL.Path == "/page":
	case strings.HasPrefix(r.URL.Path, "/page/"):
		slug = strings.Trim(strings.TrimPrefix(r.URL.Path, "/page/"), "/")
	default:
		http.NotFound(w, r)
		return
	}

	page, ok := site.pageBySlug(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, pageContext{Page: page}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *appServer) serveEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := s.hub.subscribe()
	defer s.hub.unsubscribe(ch)

	_, _ = io.WriteString(w, "retry: 1000\n\n")
	flusher.Flush()

	keepAlive := time.NewTicker(20 * time.Second)
	defer keepAlive.Stop()
	for {
		select {
		case payload := <-ch:
			_, _ = fmt.Fprintf(w, "event: reload\ndata: %s\n\n", payload)
			flusher.Flush()
		case <-keepAlive.C:
			_, _ = io.WriteString(w, ": keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *appServer) serveHealth(w http.ResponseWriter, _ *http.Request) {
	status := s.healthStatus()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if status.Status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(status)
}

func (s *appServer) serveLocalAsset(w http.ResponseWriter, r *http.Request) {
	site := s.currentSite()
	if site == nil {
		http.NotFound(w, r)
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/_local-assets/"), "/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	path, ok := site.LocalAssets[id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func (s *appServer) serveFaviconCache(w http.ResponseWriter, r *http.Request) {
	site := s.currentSite()
	if site == nil {
		http.NotFound(w, r)
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/_favicon-cache/"), "/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	targetURL, ok := site.FaviconTargets[id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if path, ok := findCachedFavicon(s.options.FaviconCacheDir, id); ok {
		http.ServeFile(w, r, path)
		return
	}
	if s.options.AssetMode == assetModeOffline {
		http.NotFound(w, r)
		return
	}
	path, err := fetchAndCacheFavicon(s.options.FaviconCacheDir, targetURL)
	if err != nil {
		log.Printf("favicon cache fetch failed for %s: %v", targetURL, err)
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func (s *appServer) reload() error {
	attemptedAt := time.Now().UTC()
	site, err := loadSite(s.configPath, s.publicDir, s.options)
	if err != nil {
		s.mu.Lock()
		s.lastReloadAttempt = attemptedAt
		s.lastReloadError = err.Error()
		s.reloadFailureCount++
		s.mu.Unlock()
		log.Printf("reload failed for %s: %v", s.configPath, err)
		return err
	}
	s.mu.Lock()
	s.site = site
	s.lastReloadAttempt = attemptedAt
	s.lastSuccessfulReload = attemptedAt
	s.lastReloadError = ""
	s.reloadSuccessCount++
	s.mu.Unlock()
	log.Printf("reload succeeded for %s (%d pages, %d tracked files)", s.configPath, len(site.Pages), len(site.TrackedFiles))
	for _, warning := range site.Warnings {
		log.Printf("config warning: %s", warning)
	}
	for _, remote := range site.RemoteAssets {
		log.Printf("remote asset enabled: %s", remote)
	}
	if site.HasRemote {
		log.Printf("remote config sources detected: file polling watch is disabled for those sources")
	}
	return nil
}

func (s *appServer) currentSite() *siteData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.site
}

func (s *appServer) trackedFiles() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.site == nil {
		return nil
	}
	out := make([]string, len(s.site.TrackedFiles))
	copy(out, s.site.TrackedFiles)
	return out
}

func (s *appServer) watch(interval time.Duration) {
	if interval <= 0 {
		log.Printf("watch disabled: interval <= 0")
		return
	}
	if s.hasRemoteSources() {
		log.Printf("watch disabled: remote config sources are not polled for changes")
		return
	}
	last := snapshotFiles(s.trackedFiles())
	if len(last) == 0 {
		log.Printf("watch disabled: no local tracked files")
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		current := snapshotFiles(s.trackedFiles())
		if fileStatesEqual(last, current) {
			continue
		}
		if err := s.reload(); err != nil {
			last = current
			continue
		}
		last = snapshotFiles(s.trackedFiles())
		s.hub.broadcast(time.Now().UTC().Format(time.RFC3339Nano))
	}
}

func (s *appServer) hasRemoteSources() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.site != nil && s.site.HasRemote
}

type healthPayload struct {
	Status               string   `json:"status"`
	Version              string   `json:"version"`
	LastReloadAttempt    string   `json:"lastReloadAttempt,omitempty"`
	LastSuccessfulReload string   `json:"lastSuccessfulReload,omitempty"`
	LastReloadError      string   `json:"lastReloadError,omitempty"`
	Warnings             []string `json:"warnings,omitempty"`
	RemoteAssets         []string `json:"remoteAssets,omitempty"`
	UnsafeHTML           bool     `json:"unsafeHtml"`
	UnsafeCSS            bool     `json:"unsafeCss"`
	ReloadSuccessCount   int      `json:"reloadSuccessCount"`
	ReloadFailureCount   int      `json:"reloadFailureCount"`
	ActiveSSEClients     int      `json:"activeSseClients"`
	TrackedFiles         int      `json:"trackedFiles"`
	Pages                int      `json:"pages"`
	RemoteSources        bool     `json:"remoteSources"`
}

func (s *appServer) healthStatus() healthPayload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	payload := healthPayload{
		Version:    appVersion,
		UnsafeHTML: s.options.AllowUnsafeHTML,
		UnsafeCSS:  s.options.AllowUnsafeCSS,
	}
	if !s.lastReloadAttempt.IsZero() {
		payload.LastReloadAttempt = s.lastReloadAttempt.Format(time.RFC3339)
	}
	if !s.lastSuccessfulReload.IsZero() {
		payload.LastSuccessfulReload = s.lastSuccessfulReload.Format(time.RFC3339)
	}
	payload.LastReloadError = s.lastReloadError
	payload.ReloadSuccessCount = s.reloadSuccessCount
	payload.ReloadFailureCount = s.reloadFailureCount
	payload.ActiveSSEClients = s.hub.count()

	if s.site != nil {
		payload.Pages = len(s.site.Pages)
		payload.TrackedFiles = len(s.site.TrackedFiles)
		payload.Warnings = append(payload.Warnings, s.site.Warnings...)
		payload.RemoteAssets = append(payload.RemoteAssets, s.site.RemoteAssets...)
		payload.RemoteSources = s.site.HasRemote
	}

	switch {
	case s.site == nil:
		payload.Status = "unavailable"
	case s.lastReloadError != "":
		payload.Status = "degraded"
	default:
		payload.Status = "ok"
	}
	return payload
}
