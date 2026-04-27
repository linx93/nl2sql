package audit

import "testing"

func TestAuditRepositoryBuildsInsertForResolvedPlanAndSQL(t *testing.T) {
	entry := loadAuditEntryFixture()

	query, args := BuildInsert(entry)
	if !contains(query, "INSERT INTO nl2sql_query_log") {
		t.Fatalf("expected insert statement, got %q", query)
	}
	if len(args) == 0 {
		t.Fatalf("expected insert args to be populated")
	}
}

func TestBuildInsertUsesNullForEmptyJSONColumns(t *testing.T) {
	entry := Entry{
		RequestID:       "req-001",
		UserID:          "user-001",
		UserRole:        "analyst",
		Domain:          "ride_hailing",
		DatasourceID:    "ride_hailing_ro",
		QueryMode:       "aggregate_overview",
		ResultKind:      "aggregate",
		ExecutionStatus: "success",
	}

	_, args := BuildInsert(entry)

	if args[6] != nil {
		t.Fatalf("expected raw_plan_json arg to be nil, got %#v", args[6])
	}
	if args[7] != nil {
		t.Fatalf("expected resolved_plan_json arg to be nil, got %#v", args[7])
	}
	if args[17] != nil {
		t.Fatalf("expected result_columns_json arg to be nil, got %#v", args[17])
	}
	if args[18] != nil {
		t.Fatalf("expected result_preview_json arg to be nil, got %#v", args[18])
	}
}

func loadAuditEntryFixture() Entry {
	return Entry{
		RequestID:           "req-001",
		UserID:              "user-123",
		UserRole:            "analyst",
		Domain:              "ride_hailing",
		DatasourceID:        "ride_hailing_ro",
		NaturalLanguageQuery: "最近30天取消率最高的前5个城市",
		QueryMode:           "ranking",
		ResultKind:          "aggregate",
		ExecutionStatus:     "success",
		ErrorCode:           "",
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
