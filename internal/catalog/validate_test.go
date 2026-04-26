package catalog

import "testing"

func TestCatalogValidationRejectsUnknownMetricColumn(t *testing.T) {
	catalog := loadBrokenCatalogFixture(t)

	err := Validate(catalog)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if want := "unknown column"; err != nil && !contains(err.Error(), want) {
		t.Fatalf("expected error to contain %q, got %q", want, err.Error())
	}
}

func loadBrokenCatalogFixture(t *testing.T) Catalog {
	t.Helper()

	catalog, err := Load("testdata/broken_catalog")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	return catalog
}

func contains(text string, want string) bool {
	return len(want) == 0 || (len(text) >= len(want) && index(text, want) >= 0)
}

func index(text string, want string) int {
	for i := 0; i+len(want) <= len(text); i++ {
		if text[i:i+len(want)] == want {
			return i
		}
	}
	return -1
}
