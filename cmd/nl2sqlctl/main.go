package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nl2sql/internal/catalog"
	"nl2sql/internal/config"
	"nl2sql/internal/scaffold"
	"nl2sql/internal/schema"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: nl2sqlctl <config|schema|scaffold> ...")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "config":
		runConfig()
	case "schema":
		runSchema()
	case "scaffold":
		runScaffold()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(2)
	}
}

func runConfig() {
	if len(os.Args) < 3 || os.Args[2] != "validate" {
		fmt.Fprintln(os.Stderr, "usage: nl2sqlctl config validate")
		os.Exit(2)
	}

	if _, err := config.LoadFromDir("configs"); err != nil {
		fmt.Fprintf(os.Stderr, "config load failed: %v\n", err)
		os.Exit(1)
	}
	cat, err := catalog.Load("configs")
	if err != nil {
		fmt.Fprintf(os.Stderr, "catalog load failed: %v\n", err)
		os.Exit(1)
	}
	if err := catalog.Validate(cat); err != nil {
		fmt.Fprintf(os.Stderr, "catalog validate failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("config validation passed")
}

func runSchema() {
	if len(os.Args) < 4 || os.Args[2] != "pull" {
		fmt.Fprintln(os.Stderr, "usage: nl2sqlctl schema pull --datasource <id>")
		os.Exit(2)
	}

	var datasourceID string
	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--datasource" && i+1 < len(os.Args) {
			datasourceID = os.Args[i+1]
			break
		}
	}
	if datasourceID == "" {
		fmt.Fprintln(os.Stderr, "schema pull requires --datasource")
		os.Exit(2)
	}

	path := filepath.Join("configs", "schemas", datasourceID+".generated.yaml")
	if _, err := schema.LoadSnapshot(path); err != nil {
		fmt.Fprintf(os.Stderr, "schema snapshot load failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("schema snapshot is available at %s\n", path)
}

func runScaffold() {
	if len(os.Args) < 6 || os.Args[2] != "domain" {
		fmt.Fprintln(os.Stderr, "usage: nl2sqlctl scaffold domain --domain <name> --datasource <id> --tables <a,b>")
		os.Exit(2)
	}

	var domainID string
	var datasourceID string
	var tablesArg string
	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--domain" && i+1 < len(os.Args) {
			domainID = os.Args[i+1]
		}
		if os.Args[i] == "--datasource" && i+1 < len(os.Args) {
			datasourceID = os.Args[i+1]
		}
		if os.Args[i] == "--tables" && i+1 < len(os.Args) {
			tablesArg = os.Args[i+1]
		}
	}

	if domainID == "" || datasourceID == "" || tablesArg == "" {
		fmt.Fprintln(os.Stderr, "scaffold domain requires --domain, --datasource and --tables")
		os.Exit(2)
	}

	snapshot, err := schema.LoadSnapshot(filepath.Join("configs", "schemas", datasourceID+".generated.yaml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "load schema snapshot failed: %v\n", err)
		os.Exit(1)
	}

	files := scaffold.ScaffoldDomain(snapshot, domainID, strings.Split(tablesArg, ","))
	for name, content := range files {
		fmt.Printf("=== %s ===\n%s\n", name, content)
	}
}
