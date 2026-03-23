package app

import "testing"

func TestUITextForSupportedLanguages(t *testing.T) {
	fr := uiTextFor("fr")
	if fr.SearchLabel != "Rechercher" || fr.NoResults != "Aucun resultat." {
		t.Fatalf("unexpected french ui text: %#v", fr)
	}

	en := uiTextFor("en")
	if en.SearchLabel != "Search" || en.NoSections != "No section is defined in this YAML." {
		t.Fatalf("unexpected english ui text: %#v", en)
	}
}
