package app

import (
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func themeCSS(name string) template.CSS {
	vars := defaultThemeVars()
	for key, value := range themeOverrides(name) {
		vars[key] = value
	}

	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(":root{")
	for _, key := range keys {
		b.WriteString("--")
		b.WriteString(key)
		b.WriteByte(':')
		b.WriteString(vars[key])
		b.WriteByte(';')
	}
	b.WriteString("}")
	return template.CSS(b.String())
}

func themeFontsURL(publicDir string) string {
	if publicDir == "" {
		return ""
	}
	path := filepath.Join(publicDir, "theme-fonts.css")
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return "/_assets/theme-fonts.css"
}

func defaultThemeVars() map[string]string {
	out := make(map[string]string, len(defaultThemeVarsData))
	for key, value := range defaultThemeVarsData {
		out[key] = value
	}
	return out
}

func themeOverrides(name string) map[string]string {
	out, ok := themeOverridesData[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil
	}
	cloned := make(map[string]string, len(out))
	for key, value := range out {
		cloned[key] = value
	}
	return cloned
}

func themeExists(name string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	return trimmed == "" || trimmed == "default" || themeOverridesData[trimmed] != nil
}
