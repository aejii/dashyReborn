package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func writeTempConfigInDir(t *testing.T, dir, name, contents string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config %s: %v", name, err)
	}
	return path
}

func writeTempFileInDir(t *testing.T, dir, name, contents string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write file %s: %v", name, err)
	}
	return path
}

func TestReadConfigStrictRejectsUnknownField(t *testing.T) {
	path := writeTempConfig(t, `
pageInfo:
  title: Demo
  unexpected: true
`)

	_, _, _, _, _, err := readConfig(path, "", loadOptions{
		Strict:    true,
		AssetMode: assetModeAuto,
	})
	if err == nil {
		t.Fatalf("expected strict validation error")
	}
	if !strings.Contains(err.Error(), "unknown field pageInfo.unexpected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadConfigWarningsIncludeIgnoredAndUnknownFields(t *testing.T) {
	path := writeTempConfig(t, `
appConfig:
  statusCheck: true
  unknownThing: true
sections:
  - name: Services
    items:
      - title: Grafana
        url: http://example.com
        id: grafana
`)

	_, _, _, warnings, _, err := readConfig(path, "", loadOptions{AssetMode: assetModeAuto})
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	expected := []string{
		"ignored field appConfig.statusCheck",
		"ignored field sections[0].items[0].id",
		"unknown field appConfig.unknownThing",
	}
	for _, want := range expected {
		if !containsWarning(warnings, want) {
			t.Fatalf("missing warning %q in %v", want, warnings)
		}
	}
}

func TestResolveConfigReferenceSupportsRemoteRelativePages(t *testing.T) {
	got := resolveConfigReference("https://example.com/dashy/configs/", "page-two.yml")
	want := "https://example.com/dashy/configs/page-two.yml"
	if got != want {
		t.Fatalf("resolveConfigReference() = %q, want %q", got, want)
	}
}

func TestReadConfigStrictRejectsMissingRequiredFields(t *testing.T) {
	path := writeTempConfig(t, `
sections:
  - items:
      - title: Grafana
`)

	_, _, _, _, _, err := readConfig(path, "", loadOptions{
		Strict:    true,
		AssetMode: assetModeAuto,
	})
	if err == nil {
		t.Fatalf("expected strict validation error")
	}
	if !strings.Contains(err.Error(), "missing required field sections[0].name") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "missing required field sections[0].items[0].url") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadConfigWarningsIncludeSemanticIssues(t *testing.T) {
	path := writeTempConfig(t, `
appConfig:
  defaultOpeningMethod: portal
pageInfo:
  navLinks:
    - title:
      path:
sections:
  - items:
      - url:
        subItems:
          - title:
            url:
  - name: Widgets
    widgets:
      - label: Clock
`)

	_, _, _, warnings, _, err := readConfig(path, "", loadOptions{AssetMode: assetModeAuto})
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	expected := []string{
		`invalid appConfig.defaultOpeningMethod value "portal", falling back to newtab`,
		`missing required field pageInfo.navLinks[0].title, rendered as "Link"`,
		`missing required field pageInfo.navLinks[0].path, rendered as "#"`,
		`missing required field sections[0].name, rendered as "Section"`,
		`missing required field sections[0].items[0].title, rendered as "Untitled"`,
		`missing required field sections[0].items[0].url, rendered as "#"`,
		`missing required field sections[0].items[0].subItems[0].title, rendered as "Link"`,
		`missing required field sections[0].items[0].subItems[0].url, rendered as "#"`,
		`missing required field sections[1].widgets[0].type, rendered as "unknown"`,
	}
	for _, want := range expected {
		if !containsWarning(warnings, want) {
			t.Fatalf("missing warning %q in %v", want, warnings)
		}
	}
}

func TestReadConfigWarningsIncludeInvalidStyleValues(t *testing.T) {
	path := writeTempConfig(t, `
sections:
  - name: Services
    items:
      - title: Grafana
        url: http://example.com
        color: red;display:none
        backgroundColor: url(https://example.com/x)
`)

	_, _, _, warnings, _, err := readConfig(path, "", loadOptions{AssetMode: assetModeAuto})
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	expected := []string{
		`invalid sections[0].items[0].color value "red;display:none", ignored`,
		`invalid sections[0].items[0].backgroundColor value "url(https://example.com/x)", ignored`,
	}
	for _, want := range expected {
		if !containsWarning(warnings, want) {
			t.Fatalf("missing warning %q in %v", want, warnings)
		}
	}
}

func containsWarning(warnings []string, want string) bool {
	for _, warning := range warnings {
		if warning == want {
			return true
		}
	}
	return false
}
