package app

import (
	"fmt"
	neturl "net/url"
	"regexp"
	"strings"
)

var cssColorPattern = regexp.MustCompile(`^(#[0-9a-fA-F]{3,8}|(?:rgb|hsl)a?\([0-9a-zA-Z\s.,/%+\-]+\)|var\(--[a-zA-Z0-9\-]+\)|[a-zA-Z]+)$`)

func loadSite(configPath, publicDir string, opts loadOptions) (*siteData, error) {
	root, tracked, baseDir, warnings, hasRemote, err := readConfig(configPath, "", opts)
	if err != nil {
		return nil, err
	}
	assets := newAssetRegistry(publicDir, opts.FaviconCacheDir, opts.AssetMode)

	rootName := fallback(root.PageInfo.Title, "Dashboard")
	pages := []pageView{buildPage(rootName, "", root, root.PageInfo, root.AppConfig, baseDir, assets, opts)}
	tracked = append([]string{}, tracked...)
	used := map[string]struct{}{"": {}}

	for _, entry := range root.Pages {
		if strings.TrimSpace(entry.Path) == "" {
			continue
		}
		ref := resolveConfigReference(baseDir, entry.Path)
		pageCfg, pageTracked, pageBaseDir, pageWarnings, pageRemote, err := readConfig(ref, baseDir, opts)
		if err != nil {
			return nil, fmt.Errorf("loading page %q: %w", entry.Name, err)
		}
		hasRemote = hasRemote || pageRemote
		tracked = append(tracked, pageTracked...)
		name := fallback(strings.TrimSpace(entry.Name), fallback(pageCfg.PageInfo.Title, "Page"))
		slug := uniqueSlug(slugify(name), used)
		for _, warning := range pageWarnings {
			warnings = append(warnings, fmt.Sprintf("page %q: %s", name, warning))
		}
		info := mergePageInfo(root.PageInfo, pageCfg.PageInfo)
		if strings.TrimSpace(info.Title) == "" {
			info.Title = name
		}
		app := mergeAppConfig(root.AppConfig, pageCfg.AppConfig)
		pages = append(pages, buildPage(name, slug, pageCfg, info, app, pageBaseDir, assets, opts))
	}

	tabs := make([]pageTab, 0, len(pages))
	for _, page := range pages {
		path := "/"
		if page.Slug != "" {
			path = "/page/" + page.Slug
		}
		tabs = append(tabs, pageTab{Name: page.Name, Path: path})
	}

	index := make(map[string]int, len(pages))
	for i := range pages {
		pages[i].Tabs = make([]pageTab, len(tabs))
		copy(pages[i].Tabs, tabs)
		pages[i].Tabs[i].Current = true
		index[pages[i].Slug] = i
	}

	return &siteData{
		Pages:          pages,
		PagesBySlug:    index,
		FaviconTargets: assets.faviconEntries(),
		LocalAssets:    assets.snapshot(),
		RemoteAssets:   assets.remoteAssets(),
		TrackedFiles:   uniqueStrings(tracked),
		Warnings:       uniqueStrings(warnings),
		HasRemote:      hasRemote,
	}, nil
}

func buildPage(name, slug string, cfg dashboardConfig, info pageInfo, app appConfig, baseDir string, assets *assetRegistry, opts loadOptions) pageView {
	sections := make([]sectionView, 0, len(cfg.Sections))
	usedSectionIDs := make(map[string]struct{}, len(cfg.Sections))
	loadFontAwesome := false
	loadMdi := false
	language := normalizeLanguage(app.Language)

	for idx, sec := range cfg.Sections {
		sectionIconURL, sectionIconClass, sectionFallback, sectionFA, sectionMDI, _ := assets.resolveIcon(sec.Icon, "", baseDir)
		loadFontAwesome = loadFontAwesome || sectionFA
		loadMdi = loadMdi || sectionMDI
		sectionID := uniqueSlug(slugify(fmt.Sprintf("%s-%d", sec.Name, idx+1)), usedSectionIDs)
		sv := sectionView{
			ID:        sectionID,
			Name:      fallback(sec.Name, "Section"),
			Marker:    fallback(sectionFallback, markerFor(sec.Name, sec.Icon)),
			IconURL:   sectionIconURL,
			IconClass: sectionIconClass,
		}

		for _, it := range sec.Items {
			target, rel := resolveTarget(it.Target, app.DefaultOpeningMethod)
			iconURL, iconClass, fallbackIcon, itemFA, itemMDI, iconIsFavicon := assets.resolveIcon(it.Icon, it.URL, baseDir)
			loadFontAwesome = loadFontAwesome || itemFA
			loadMdi = loadMdi || itemMDI
			subViews := make([]subItemView, 0, len(it.SubItems))
			searchParts := []string{it.Title, it.Description, it.Provider}

			for _, sub := range it.SubItems {
				subTarget, subRel := resolveTarget(sub.Target, app.DefaultOpeningMethod)
				subViews = append(subViews, subItemView{
					Title:  fallback(sub.Title, "Link"),
					URL:    fallback(sub.URL, "#"),
					Target: subTarget,
					Rel:    subRel,
				})
				searchParts = append(searchParts, sub.Title)
			}

			sv.Items = append(sv.Items, itemView{
				Title:         fallback(it.Title, "Untitled"),
				Description:   it.Description,
				URL:           fallback(it.URL, "#"),
				Target:        target,
				Rel:           rel,
				Provider:      it.Provider,
				IconURL:       iconURL,
				IconIsFavicon: iconIsFavicon,
				IconClass:     iconClass,
				FallbackIcon:  fallbackIcon,
				Style:         cardStyle(it.Color, it.BackgroundColor),
				SearchText:    strings.ToLower(strings.Join(searchParts, " ")),
				SubItems:      subViews,
			})
		}

		for _, w := range sec.Widgets {
			label := fallback(strings.TrimSpace(w.Label), fallback(w.Type, "Widget"))
			sv.Widgets = append(sv.Widgets, widgetView{
				Label:        label,
				Type:         fallback(w.Type, "unknown"),
				FallbackIcon: markerFor(label, ""),
				SearchText:   strings.ToLower(label + " " + w.Type),
			})
		}

		sections = append(sections, sv)
	}

	allowCustomCSS := opts.AllowUnsafeCSS && strings.TrimSpace(app.CustomCSS) != ""
	allowUnsafeHTML := opts.AllowUnsafeHTML && strings.TrimSpace(info.FooterText) != ""
	if loadFontAwesome {
		assets.noteRemote(fontAwesomeCSSURL)
	}
	if loadMdi {
		assets.noteRemote(mdiCSSURL)
	}

	return pageView{
		Language:        language,
		UI:              uiTextFor(language),
		Name:            name,
		Slug:            slug,
		Title:           fallback(info.Title, name),
		Description:     info.Description,
		Logo:            assets.resolve(info.Logo, baseDir),
		BackgroundImage: assets.resolve(app.BackgroundImg, baseDir),
		SectionGrid:     sectionGrid(app.Layout, app.ColCount),
		ItemSize:        normalizeItemSize(app.IconSize),
		ThemeFontsURL:   themeFontsURL(assets.publicDir),
		ThemeCSS:        themeCSS(app.Theme),
		CustomCSS:       templateCSS(app.CustomCSS),
		AllowCustomCSS:  allowCustomCSS,
		LoadFontAwesome: loadFontAwesome,
		LoadMdi:         loadMdi,
		AllowUnsafeHTML: allowUnsafeHTML,
		FooterHTML:      templateHTML(info.FooterText),
		FooterText:      info.FooterText,
		NavLinks:        buildNavLinks(info.NavLinks),
		Sections:        sections,
	}
}

func buildNavLinks(in []navLink) []navLinkView {
	out := make([]navLinkView, 0, len(in))
	for _, link := range in {
		target, rel := resolveTarget(link.Target, "newtab")
		out = append(out, navLinkView{
			Title:  fallback(link.Title, "Link"),
			Path:   fallback(link.Path, "#"),
			Target: target,
			Rel:    rel,
		})
	}
	return out
}

func (s *siteData) pageBySlug(slug string) (pageView, bool) {
	i, ok := s.PagesBySlug[slug]
	if !ok || i >= len(s.Pages) {
		return pageView{}, false
	}
	return s.Pages[i], true
}

func resolveTarget(raw, fallbackMethod string) (string, string) {
	method := strings.ToLower(strings.TrimSpace(raw))
	if method == "" {
		method = strings.ToLower(strings.TrimSpace(fallbackMethod))
	}
	switch method {
	case "sametab":
		return "_self", ""
	case "parent":
		return "_parent", ""
	case "top":
		return "_top", ""
	default:
		return "_blank", "noreferrer noopener"
	}
}

func normalizeFontAwesomeClass(icon string) string {
	icon = strings.TrimSpace(icon)
	if strings.Contains(icon, " ") {
		return icon
	}
	if strings.HasPrefix(icon, "fa-") {
		return "fa-solid " + icon
	}
	return icon
}

func queryEscape(raw string) string {
	return neturl.QueryEscape(raw)
}

func cardStyle(color, background string) string {
	var styles []string
	if background = safeColorValue(background); background != "" {
		styles = append(styles, "--item-background:"+background)
		styles = append(styles, "--item-background-hover:"+background)
	}
	if color = safeColorValue(color); color != "" {
		styles = append(styles, "--item-text-color:"+color)
		styles = append(styles, "--item-text-color-hover:"+color)
	}
	return strings.Join(styles, ";")
}

func safeColorValue(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	lowered := strings.ToLower(value)
	if strings.ContainsAny(value, `;{}<>"'`) || strings.Contains(lowered, "url(") || strings.Contains(lowered, "expression(") {
		return ""
	}
	if !cssColorPattern.MatchString(value) {
		return ""
	}
	return value
}
