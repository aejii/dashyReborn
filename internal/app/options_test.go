package app

import "testing"

func TestParseAssetMode(t *testing.T) {
	tests := map[string]assetMode{
		"":              assetModeAuto,
		"auto":          assetModeAuto,
		"internal-only": assetModeInternalOnly,
		"offline":       assetModeOffline,
	}
	for raw, want := range tests {
		got, err := parseAssetMode(raw)
		if err != nil {
			t.Fatalf("parseAssetMode(%q) unexpected error: %v", raw, err)
		}
		if got != want {
			t.Fatalf("parseAssetMode(%q) = %q, want %q", raw, got, want)
		}
	}
	if _, err := parseAssetMode("invalid"); err == nil {
		t.Fatalf("expected invalid mode to return an error")
	}
}
