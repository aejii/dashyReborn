package app

import (
	"embed"
	"html/template"
)

//go:embed page.gohtml page.css page.js
var templateFS embed.FS

//go:embed page.css
var embeddedPageCSS string

//go:embed page.js
var embeddedPageJS string

func newTemplate() (*template.Template, error) {
	return template.New("page.gohtml").Funcs(template.FuncMap{
		"pageCSS":           func() template.CSS { return template.CSS(embeddedPageCSS) },
		"pageJS":            func() template.JS { return template.JS(embeddedPageJS) },
		"fontAwesomeCSSURL": func() string { return fontAwesomeCSSURL },
		"mdiCSSURL":         func() string { return mdiCSSURL },
	}).ParseFS(templateFS, "page.gohtml")
}

func templateCSS(value string) template.CSS {
	return template.CSS(value)
}

func templateHTML(value string) template.HTML {
	return template.HTML(value)
}
