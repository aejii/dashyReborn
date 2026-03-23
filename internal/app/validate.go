package app

import (
	"fmt"
	"sort"
	"strings"
)

func configWarnings(cfg dashboardConfig) []string {
	var warnings []string

	appendMapWarnings(&warnings, "dashboard", cfg.Extra, nil)
	appendMapWarnings(&warnings, "pageInfo", cfg.PageInfo.Extra, nil)
	appendMapWarnings(&warnings, "appConfig", cfg.AppConfig.Extra, map[string]string{
		"customColors":         "ignored",
		"enableErrorReporting": "ignored",
		"statusCheck":          "ignored",
	})

	for i, link := range cfg.PageInfo.NavLinks {
		appendMapWarnings(&warnings, fmt.Sprintf("pageInfo.navLinks[%d]", i), link.Extra, nil)
	}
	for i, page := range cfg.Pages {
		appendMapWarnings(&warnings, fmt.Sprintf("pages[%d]", i), page.Extra, nil)
	}
	for i, sec := range cfg.Sections {
		appendMapWarnings(&warnings, fmt.Sprintf("sections[%d]", i), sec.Extra, nil)
		for j, item := range sec.Items {
			appendMapWarnings(&warnings, fmt.Sprintf("sections[%d].items[%d]", i, j), item.Extra, map[string]string{
				"id": "ignored",
			})
			if strings.TrimSpace(item.Color) != "" && safeColorValue(item.Color) == "" {
				warnings = append(warnings, fmt.Sprintf("invalid sections[%d].items[%d].color value %q, ignored", i, j, item.Color))
			}
			if strings.TrimSpace(item.BackgroundColor) != "" && safeColorValue(item.BackgroundColor) == "" {
				warnings = append(warnings, fmt.Sprintf("invalid sections[%d].items[%d].backgroundColor value %q, ignored", i, j, item.BackgroundColor))
			}
			for k, sub := range item.SubItems {
				appendMapWarnings(&warnings, fmt.Sprintf("sections[%d].items[%d].subItems[%d]", i, j, k), sub.Extra, nil)
			}
		}
		for j, widget := range sec.Widgets {
			appendMapWarnings(&warnings, fmt.Sprintf("sections[%d].widgets[%d]", i, j), widget.Extra, nil)
		}
	}

	if strings.TrimSpace(cfg.AppConfig.Layout) != "" {
		switch strings.ToLower(strings.TrimSpace(cfg.AppConfig.Layout)) {
		case "auto", "vertical", "single-column", "singlecolumn", "grid":
		default:
			warnings = append(warnings, fmt.Sprintf("unknown appConfig.layout value %q, falling back to auto", cfg.AppConfig.Layout))
		}
	}

	if strings.TrimSpace(cfg.AppConfig.Theme) != "" && !themeExists(cfg.AppConfig.Theme) {
		warnings = append(warnings, fmt.Sprintf("unknown appConfig.theme value %q, using default theme variables", cfg.AppConfig.Theme))
	}
	if method := strings.ToLower(strings.TrimSpace(cfg.AppConfig.DefaultOpeningMethod)); method != "" {
		switch method {
		case "newtab", "sametab", "parent", "top":
		default:
			warnings = append(warnings, fmt.Sprintf("invalid appConfig.defaultOpeningMethod value %q, falling back to newtab", cfg.AppConfig.DefaultOpeningMethod))
		}
	}

	appendSemanticWarnings(&warnings, cfg)

	sort.Strings(warnings)
	return warnings
}

func appendSemanticWarnings(out *[]string, cfg dashboardConfig) {
	for i, link := range cfg.PageInfo.NavLinks {
		if strings.TrimSpace(link.Title) == "" {
			*out = append(*out, fmt.Sprintf("missing required field pageInfo.navLinks[%d].title, rendered as \"Link\"", i))
		}
		if strings.TrimSpace(link.Path) == "" {
			*out = append(*out, fmt.Sprintf("missing required field pageInfo.navLinks[%d].path, rendered as \"#\"", i))
		}
	}
	for i, page := range cfg.Pages {
		if strings.TrimSpace(page.Path) == "" {
			*out = append(*out, fmt.Sprintf("missing required field pages[%d].path", i))
		}
	}
	for i, sec := range cfg.Sections {
		if strings.TrimSpace(sec.Name) == "" {
			*out = append(*out, fmt.Sprintf("missing required field sections[%d].name, rendered as \"Section\"", i))
		}
		for j, item := range sec.Items {
			if strings.TrimSpace(item.Title) == "" {
				*out = append(*out, fmt.Sprintf("missing required field sections[%d].items[%d].title, rendered as \"Untitled\"", i, j))
			}
			if strings.TrimSpace(item.URL) == "" {
				*out = append(*out, fmt.Sprintf("missing required field sections[%d].items[%d].url, rendered as \"#\"", i, j))
			}
			for k, sub := range item.SubItems {
				if strings.TrimSpace(sub.Title) == "" {
					*out = append(*out, fmt.Sprintf("missing required field sections[%d].items[%d].subItems[%d].title, rendered as \"Link\"", i, j, k))
				}
				if strings.TrimSpace(sub.URL) == "" {
					*out = append(*out, fmt.Sprintf("missing required field sections[%d].items[%d].subItems[%d].url, rendered as \"#\"", i, j, k))
				}
			}
		}
		for j, widget := range sec.Widgets {
			if strings.TrimSpace(widget.Type) == "" {
				*out = append(*out, fmt.Sprintf("missing required field sections[%d].widgets[%d].type, rendered as \"unknown\"", i, j))
			}
		}
	}
}

func appendMapWarnings(out *[]string, scope string, extras map[string]any, known map[string]string) {
	if len(extras) == 0 {
		return
	}

	keys := make([]string, 0, len(extras))
	for key := range extras {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if state, ok := known[key]; ok {
			*out = append(*out, fmt.Sprintf("%s field %s.%s", state, scope, key))
			continue
		}
		*out = append(*out, fmt.Sprintf("unknown field %s.%s", scope, key))
	}
}
