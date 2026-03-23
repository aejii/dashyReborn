package app

import "html/template"

type dashboardConfig struct {
	PageInfo  pageInfo       `yaml:"pageInfo"`
	AppConfig appConfig      `yaml:"appConfig"`
	Sections  []section      `yaml:"sections"`
	Pages     []pageEntry    `yaml:"pages"`
	Extra     map[string]any `yaml:",inline"`
}

type pageInfo struct {
	Title       string         `yaml:"title"`
	Description string         `yaml:"description"`
	FooterText  string         `yaml:"footerText"`
	Logo        string         `yaml:"logo"`
	NavLinks    []navLink      `yaml:"navLinks"`
	Extra       map[string]any `yaml:",inline"`
}

type navLink struct {
	Title  string         `yaml:"title"`
	Path   string         `yaml:"path"`
	Target string         `yaml:"target"`
	Extra  map[string]any `yaml:",inline"`
}

type appConfig struct {
	Theme                string         `yaml:"theme"`
	DefaultOpeningMethod string         `yaml:"defaultOpeningMethod"`
	BackgroundImg        string         `yaml:"backgroundImg"`
	CustomCSS            string         `yaml:"customCss"`
	ColCount             int            `yaml:"colCount"`
	IconSize             string         `yaml:"iconSize"`
	Layout               string         `yaml:"layout"`
	Language             string         `yaml:"language"`
	Extra                map[string]any `yaml:",inline"`
}

type pageEntry struct {
	Name  string         `yaml:"name"`
	Path  string         `yaml:"path"`
	Extra map[string]any `yaml:",inline"`
}

type section struct {
	Name    string         `yaml:"name"`
	Icon    string         `yaml:"icon"`
	Items   []item         `yaml:"items"`
	Widgets []widget       `yaml:"widgets"`
	Extra   map[string]any `yaml:",inline"`
}

type item struct {
	Title           string         `yaml:"title"`
	Description     string         `yaml:"description"`
	URL             string         `yaml:"url"`
	Icon            string         `yaml:"icon"`
	Target          string         `yaml:"target"`
	Provider        string         `yaml:"provider"`
	Color           string         `yaml:"color"`
	BackgroundColor string         `yaml:"backgroundColor"`
	SubItems        []subItem      `yaml:"subItems"`
	Extra           map[string]any `yaml:",inline"`
}

type subItem struct {
	Title  string         `yaml:"title"`
	URL    string         `yaml:"url"`
	Target string         `yaml:"target"`
	Extra  map[string]any `yaml:",inline"`
}

type widget struct {
	Type  string         `yaml:"type"`
	Label string         `yaml:"label"`
	Extra map[string]any `yaml:",inline"`
}

type navLinkView struct {
	Title  string
	Path   string
	Target string
	Rel    string
}

type pageTab struct {
	Name    string
	Path    string
	Current bool
}

type subItemView struct {
	Title  string
	URL    string
	Target string
	Rel    string
}

type itemView struct {
	Title         string
	Description   string
	URL           string
	Target        string
	Rel           string
	Provider      string
	IconURL       string
	IconIsFavicon bool
	IconClass     string
	FallbackIcon  string
	Style         string
	SearchText    string
	SubItems      []subItemView
}

type widgetView struct {
	Label        string
	Type         string
	FallbackIcon string
	SearchText   string
}

type uiText struct {
	SearchLabel       string
	SearchPlaceholder string
	NoResults         string
	NoSections        string
	WidgetLabel       string
}

type sectionView struct {
	ID        string
	Name      string
	Marker    string
	IconURL   string
	IconClass string
	Items     []itemView
	Widgets   []widgetView
}

type pageView struct {
	Name            string
	Slug            string
	Language        string
	UI              uiText
	Title           string
	Description     string
	Logo            string
	BackgroundImage string
	SectionGrid     string
	ItemSize        string
	ThemeFontsURL   string
	ThemeCSS        template.CSS
	CustomCSS       template.CSS
	AllowCustomCSS  bool
	LoadFontAwesome bool
	LoadMdi         bool
	AllowUnsafeHTML bool
	FooterHTML      template.HTML
	FooterText      string
	NavLinks        []navLinkView
	Tabs            []pageTab
	Sections        []sectionView
}

type pageContext struct {
	Page pageView
}

type siteData struct {
	Pages          []pageView
	PagesBySlug    map[string]int
	FaviconTargets map[string]string
	LocalAssets    map[string]string
	RemoteAssets   []string
	TrackedFiles   []string
	Warnings       []string
	HasRemote      bool
}

type fileState struct {
	Exists  bool
	Size    int64
	ModTime int64
}
