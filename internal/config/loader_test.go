package config

import "testing"

func TestLoadConfigReadsDatasourcesAndDomainRouting(t *testing.T) {
	cfg, err := LoadFromDir("testdata/basic")
	if err != nil {
		t.Fatalf("LoadFromDir returned error: %v", err)
	}

	domain, ok := cfg.Domains["ride_hailing"]
	if !ok {
		t.Fatalf("expected ride_hailing domain to be loaded")
	}
	if domain.DatasourceID != "ride_hailing_ro" {
		t.Fatalf("expected datasource_id ride_hailing_ro, got %q", domain.DatasourceID)
	}

	datasource, ok := cfg.Datasources["ride_hailing_ro"]
	if !ok {
		t.Fatalf("expected ride_hailing_ro datasource to be loaded")
	}
	if datasource.Driver != "mysql" {
		t.Fatalf("expected mysql driver, got %q", datasource.Driver)
	}
}
