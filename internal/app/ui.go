package app

func uiTextFor(language string) uiText {
	switch language {
	case "fr":
		return uiText{
			SearchLabel:       "Rechercher",
			SearchPlaceholder: "Commencez a taper pour filtrer",
			NoResults:         "Aucun resultat.",
			NoSections:        "Aucune section n'est definie dans ce YAML.",
			WidgetLabel:       "Widget",
		}
	default:
		return uiText{
			SearchLabel:       "Search",
			SearchPlaceholder: "Start typing to filter",
			NoResults:         "No results.",
			NoSections:        "No section is defined in this YAML.",
			WidgetLabel:       "Widget",
		}
	}
}
