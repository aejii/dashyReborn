package app

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var appVersion = "dev"

func Run(version string) error {
	if version != "" {
		appVersion = version
	}

	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	configPath := flag.String("config", "dashy/user-data/conf.yml", "Dashy-compatible YAML config")
	publicDir := flag.String("public", "", "Dashy-style public assets directory")
	watchEvery := flag.Duration("watch", 2*time.Second, "Config polling interval, set to 0 to disable")
	strict := flag.Bool("strict", false, "Fail on unknown YAML fields")
	allowUnsafeHTML := flag.Bool("allow-unsafe-html", false, "Render footerText as raw HTML")
	allowUnsafeCSS := flag.Bool("allow-unsafe-css", false, "Render customCss as raw CSS")
	assetsModeRaw := flag.String("assets-mode", "internal-only", "Assets mode: auto, internal-only, offline")
	faviconCacheDir := flag.String("favicon-cache-dir", ".cache/favicons", "Directory used to persist remote favicons locally")
	flag.Parse()

	cfg, err := normalizeConfigPath(*configPath)
	if err != nil {
		return fmt.Errorf("resolve config: %w", err)
	}
	assetsMode, err := parseAssetMode(*assetsModeRaw)
	if err != nil {
		return fmt.Errorf("parse assets mode: %w", err)
	}

	pub := *publicDir
	if pub == "" {
		pub = detectPublicDir(cfg)
	}
	if pub != "" {
		if abs, err := filepath.Abs(pub); err == nil {
			pub = abs
		}
	}

	options := loadOptions{
		Strict:          *strict,
		AllowUnsafeHTML: *allowUnsafeHTML,
		AllowUnsafeCSS:  *allowUnsafeCSS,
		AssetMode:       assetsMode,
		FaviconCacheDir: *faviconCacheDir,
	}

	if isRemote(cfg) && *watchEvery > 0 {
		log.Printf("watch requested for remote config %s, but automatic polling reload remains disabled for remote sources", cfg)
	}
	if *allowUnsafeHTML {
		log.Printf("unsafe HTML rendering enabled for footerText")
	}
	if *allowUnsafeCSS {
		log.Printf("unsafe CSS rendering enabled for customCss")
	}

	srv, err := newServer(cfg, pub, options)
	if err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	go srv.watch(*watchEvery)

	server := &http.Server{
		Addr:              *addr,
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	log.Printf("serving %s on http://%s (version=%s, assets-mode=%s)", cfg, *addr, appVersion, assetsMode)
	if pub != "" {
		log.Printf("serving assets from %s", pub)
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	stop, stopCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopCancel()

	select {
	case <-stop.Done():
		log.Printf("shutdown requested")
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
