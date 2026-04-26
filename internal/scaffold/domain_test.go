package scaffold

import "testing"

func TestScaffoldDomainCreatesDisabledDimensionSpecsByDefault(t *testing.T) {
	files := ScaffoldDomain(loadSchemaFixture(), "ride_hailing", []string{"trip_orders"})
	content, ok := files["dimensions.yaml"]
	if !ok {
		t.Fatalf("expected dimensions.yaml to be scaffolded")
	}
	if !contains(content, "enabled: false") {
		t.Fatalf("expected scaffolded dimensions to be disabled by default, got %q", content)
	}
}

func loadSchemaFixture() SchemaSnapshot {
	return SchemaSnapshot{
		DatasourceID: "ride_hailing_ro",
		Database:     "ride_hailing",
		Tables: []Table{
			{
				Name: "trip_orders",
				Columns: []Column{
					{Name: "city_code", DataType: "varchar", Comment: "城市编码"},
				},
			},
		},
	}
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
