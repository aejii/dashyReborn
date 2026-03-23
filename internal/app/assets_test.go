package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAssetRegistryResolveRemoteModes(t *testing.T) {
	auto := newAssetRegistry("", "", assetModeAuto)
	if got := auto.resolve("https://example.com/logo.svg", ""); got != "https://example.com/logo.svg" {
		t.Fatalf("resolve() = %q", got)
	}

	offline := newAssetRegistry("", "", assetModeOffline)
	if got := offline.resolve("https://example.com/logo.svg", ""); got != "" {
		t.Fatalf("resolve() = %q, want empty string", got)
	}
}

func TestAssetRegistryResolveRelativeRemoteAsset(t *testing.T) {
	auto := newAssetRegistry("", "", assetModeAuto)
	got := auto.resolve("./logo.svg", "https://example.com/assets/")
	want := "https://example.com/assets/logo.svg"
	if got != want {
		t.Fatalf("resolve() = %q, want %q", got, want)
	}
}

func TestLooksLikeAssetReferenceAndPathWithin(t *testing.T) {
	if !looksLikeAssetReference("./icons/logo.svg") {
		t.Fatalf("expected relative path to be detected as asset reference")
	}
	if looksLikeAssetReference("grafana") {
		t.Fatalf("expected plain provider token not to be treated as asset reference")
	}

	root := t.TempDir()
	publicDir := filepath.Join(root, "public")
	target := filepath.Join(publicDir, "icons", "logo.svg")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir icons dir: %v", err)
	}
	rel, ok := pathWithin(root, target)
	if !ok || filepath.ToSlash(rel) != "public/icons/logo.svg" {
		t.Fatalf("pathWithin() = (%q, %v)", rel, ok)
	}
}
