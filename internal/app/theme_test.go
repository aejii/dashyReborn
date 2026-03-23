package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThemeCSSAndExistence(t *testing.T) {
	if !themeExists("dracula") {
		t.Fatalf("expected known theme to exist")
	}
	if themeExists("not-a-theme") {
		t.Fatalf("expected unknown theme to be rejected")
	}
	css := string(themeCSS("dracula"))
	if !strings.Contains(css, "--background:") {
		t.Fatalf("expected theme CSS to contain background variable")
	}
}

func TestThemeFontsURL(t *testing.T) {
	publicDir := t.TempDir()
	if got := themeFontsURL(publicDir); got != "" {
		t.Fatalf("themeFontsURL() = %q, want empty string", got)
	}
	path := filepath.Join(publicDir, "theme-fonts.css")
	if err := os.WriteFile(path, []byte("body{}"), 0o600); err != nil {
		t.Fatalf("write theme-fonts.css: %v", err)
	}
	if got := themeFontsURL(publicDir); got != "/_assets/theme-fonts.css" {
		t.Fatalf("themeFontsURL() = %q, want %q", got, "/_assets/theme-fonts.css")
	}
}
