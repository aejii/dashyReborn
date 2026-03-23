package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlugifyAndUniqueSlug(t *testing.T) {
	if got := slugify(" Grafana Home "); got != "grafana-home" {
		t.Fatalf("slugify() = %q, want %q", got, "grafana-home")
	}

	used := map[string]struct{}{}
	if got := uniqueSlug("grafana", used); got != "grafana" {
		t.Fatalf("uniqueSlug() = %q, want %q", got, "grafana")
	}
	if got := uniqueSlug("grafana", used); got != "grafana-2" {
		t.Fatalf("uniqueSlug() = %q, want %q", got, "grafana-2")
	}
}

func TestMarkerAndLayoutHelpers(t *testing.T) {
	if got := markerFor("mdi-home", ""); got != "H" {
		t.Fatalf("markerFor() = %q, want %q", got, "H")
	}
	if got := sectionGrid("vertical", 0); got != "1fr" {
		t.Fatalf("sectionGrid() = %q, want %q", got, "1fr")
	}
	if got := sectionGrid("", 9); got != "repeat(6,minmax(300px,1fr))" {
		t.Fatalf("sectionGrid() = %q", got)
	}
	if got := normalizeLanguage(""); got != "en" {
		t.Fatalf("normalizeLanguage() = %q, want %q", got, "en")
	}
	if got := normalizeItemSize("large"); got != "large" {
		t.Fatalf("normalizeItemSize() = %q, want %q", got, "large")
	}
}

func TestSnapshotFilesAndEquality(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte("demo"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	a := snapshotFiles([]string{path})
	b := snapshotFiles([]string{path})
	if !fileStatesEqual(a, b) {
		t.Fatalf("expected snapshots to match")
	}
}
