package catalog

import "testing"

func TestCatalogLoaderBuildsTableAndColumnIndex(t *testing.T) {
	catalog, err := Load("testdata/catalog")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if _, ok := catalog.TablesByName["trip_orders"]; !ok {
		t.Fatalf("expected trip_orders table to be indexed")
	}

	columns, ok := catalog.ColumnsByTable["trip_orders"]
	if !ok {
		t.Fatalf("expected trip_orders columns to be indexed")
	}
	if _, ok := columns["called_at"]; !ok {
		t.Fatalf("expected called_at column to be indexed")
	}
}
