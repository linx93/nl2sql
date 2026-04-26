package datasource

import (
	"database/sql"
	"testing"

	"nl2sql/internal/config"
)

func TestDatasourceRegistryReturnsPoolByDomain(t *testing.T) {
	registry := NewRegistry()
	expected := &sql.DB{}
	registry.Register("ride_hailing_ro", expected)

	db, err := registry.ForDomain("ride_hailing", map[string]config.DomainConfig{
		"ride_hailing": {
			ID:           "ride_hailing",
			DatasourceID: "ride_hailing_ro",
		},
	})
	if err != nil {
		t.Fatalf("ForDomain returned error: %v", err)
	}
	if db != expected {
		t.Fatalf("expected registered pool to be returned")
	}
}
