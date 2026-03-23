package app

import (
	"fmt"
	"strings"
)

type assetMode string

const (
	assetModeAuto         assetMode = "auto"
	assetModeInternalOnly assetMode = "internal-only"
	assetModeOffline      assetMode = "offline"
)

type loadOptions struct {
	Strict          bool
	AllowUnsafeHTML bool
	AllowUnsafeCSS  bool
	AssetMode       assetMode
	FaviconCacheDir string
}

func parseAssetMode(raw string) (assetMode, error) {
	switch assetMode(strings.ToLower(strings.TrimSpace(raw))) {
	case "", assetModeAuto:
		return assetModeAuto, nil
	case assetModeInternalOnly:
		return assetModeInternalOnly, nil
	case assetModeOffline:
		return assetModeOffline, nil
	default:
		return "", fmt.Errorf("unsupported assets mode %q", raw)
	}
}

func (m assetMode) allowsRemoteAssets() bool {
	return m == assetModeAuto
}

func (m assetMode) allowsRemoteConfig() bool {
	return true
}
