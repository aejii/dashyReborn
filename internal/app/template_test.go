package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestTemplateRendersEmbeddedAssetsAndEnglishTexts(t *testing.T) {
	assets := newAssetRegistry("", "", assetModeAuto)
	page := buildPage(
		"Demo",
		"",
		dashboardConfig{},
		pageInfo{Title: "Demo"},
		appConfig{},
		"",
		assets,
		loadOptions{AssetMode: assetModeAuto},
	)

	tmpl, err := newTemplate()
	if err != nil {
		t.Fatalf("newTemplate: %v", err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, pageContext{Page: page}); err != nil {
		t.Fatalf("execute template: %v", err)
	}

	rendered := out.String()
	if !strings.Contains(rendered, "Start typing to filter") {
		t.Fatalf("expected english placeholder in rendered template")
	}
	if !strings.Contains(rendered, "dashyreborn:collapse:") {
		t.Fatalf("expected embedded JS to be rendered")
	}
}
