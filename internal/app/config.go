package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func readConfig(ref, baseDir string, opts loadOptions) (dashboardConfig, []string, string, []string, bool, error) {
	data, tracked, actualBase, remote, err := readSource(ref, baseDir)
	if err != nil {
		return dashboardConfig{}, nil, "", nil, false, err
	}

	var cfg dashboardConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(opts.Strict)
	if err := decoder.Decode(&cfg); err != nil {
		return dashboardConfig{}, nil, "", nil, remote, err
	}

	warnings := configWarnings(cfg)
	if opts.Strict {
		if err := strictConfigError(warnings); err != nil {
			return dashboardConfig{}, nil, "", warnings, remote, err
		}
	}

	return cfg, tracked, actualBase, warnings, remote, nil
}

func readSource(ref, baseDir string) ([]byte, []string, string, bool, error) {
	if isRemote(ref) {
		client := &http.Client{Timeout: 4 * time.Second}
		resp, err := client.Get(ref)
		if err != nil {
			return nil, nil, "", true, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, nil, "", true, fmt.Errorf("remote config returned %s", resp.Status)
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, "", true, err
		}
		return data, nil, remoteBase(ref), true, nil
	}

	path := resolveConfigReference(baseDir, ref)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, "", false, err
	}
	return data, []string{path}, filepath.Dir(path), false, nil
}

func normalizeConfigPath(path string) (string, error) {
	if isRemote(path) {
		return path, nil
	}
	return filepath.Abs(path)
}

func detectPublicDir(configPath string) string {
	if isRemote(configPath) {
		return ""
	}

	dir := filepath.Dir(configPath)
	candidates := []string{filepath.Join(dir, "public")}
	if filepath.Base(dir) == "user-data" {
		candidates = append([]string{filepath.Join(filepath.Dir(dir), "public")}, candidates...)
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

func resolveConfigReference(baseDir, ref string) string {
	if isRemote(ref) || filepath.IsAbs(ref) || baseDir == "" {
		return ref
	}
	if isRemote(baseDir) {
		base, err := neturl.Parse(baseDir)
		if err == nil {
			rel, relErr := neturl.Parse(ref)
			if relErr == nil {
				return base.ResolveReference(rel).String()
			}
		}
	}
	return filepath.Clean(filepath.Join(baseDir, ref))
}

func mergePageInfo(base, override pageInfo) pageInfo {
	out := base
	if override.Title != "" {
		out.Title = override.Title
	}
	if override.Description != "" {
		out.Description = override.Description
	}
	if override.FooterText != "" {
		out.FooterText = override.FooterText
	}
	if override.Logo != "" {
		out.Logo = override.Logo
	}
	if len(override.NavLinks) > 0 {
		out.NavLinks = override.NavLinks
	}
	return out
}

func mergeAppConfig(base, override appConfig) appConfig {
	out := base
	if override.Theme != "" {
		out.Theme = override.Theme
	}
	if override.DefaultOpeningMethod != "" {
		out.DefaultOpeningMethod = override.DefaultOpeningMethod
	}
	if override.BackgroundImg != "" {
		out.BackgroundImg = override.BackgroundImg
	}
	if override.CustomCSS != "" {
		out.CustomCSS = override.CustomCSS
	}
	if override.ColCount > 0 {
		out.ColCount = override.ColCount
	}
	if override.IconSize != "" {
		out.IconSize = override.IconSize
	}
	if override.Layout != "" {
		out.Layout = override.Layout
	}
	if override.Language != "" {
		out.Language = override.Language
	}
	return out
}

func isRemote(ref string) bool {
	return strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://")
}

func remoteBase(raw string) string {
	parsed, err := neturl.Parse(raw)
	if err != nil {
		return ""
	}
	if parsed.Path == "" {
		parsed.Path = "/"
		return parsed.String()
	}
	lastSlash := strings.LastIndex(parsed.Path, "/")
	if lastSlash < 0 {
		parsed.Path = "/"
		return parsed.String()
	}
	parsed.Path = parsed.Path[:lastSlash+1]
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func strictConfigError(warnings []string) error {
	var blocking []string
	for _, warning := range warnings {
		switch {
		case strings.HasPrefix(warning, "unknown field "):
			blocking = append(blocking, warning)
		case strings.HasPrefix(warning, "missing required field "):
			blocking = append(blocking, warning)
		}
	}
	if len(blocking) == 0 {
		return nil
	}
	sort.Strings(blocking)
	return errors.New(strings.Join(blocking, "; "))
}
