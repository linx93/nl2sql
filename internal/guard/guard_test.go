package guard

import (
	"testing"

	"nl2sql/internal/builder"
	"nl2sql/internal/catalog"
	"nl2sql/internal/domain"
)

func TestGuardRejectsDetailQueryWithoutAllowedColumns(t *testing.T) {
	input := loadGuardInputWithSensitiveColumnFixture()

	_, err := Validate(input)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if err.Error() != "column not allowed: trip_orders.secret_note" {
		t.Fatalf("expected column whitelist error, got %q", err.Error())
	}
}

func loadGuardInputWithSensitiveColumnFixture() GuardInput {
	return GuardInput{
		Plan: domain.ResolvedPlan{
			QueryMode:    domain.QueryModeDetailList,
			DetailViewID: "detail.waiting_pickup_orders",
		},
		BuildResult: builder.BuildResult{
			SQL: "SELECT trip_orders.order_id, trip_orders.secret_note FROM trip_orders LIMIT ?",
			ReferencedCols: []string{
				"trip_orders.order_id",
				"trip_orders.secret_note",
			},
			ReferencedTables: []string{"trip_orders"},
			Limit:            50,
		},
		DetailView: catalog.DetailViewSpec{
			ID: "detail.waiting_pickup_orders",
			AllowedSelectColumns: []string{
				"trip_orders.order_id",
				"trip_orders.called_at",
			},
			AllowedFilterFields: []string{
				"trip_orders.city_code",
			},
			RequiredTimeField: "trip_orders.called_at",
			DefaultSort: catalog.SortSpec{
				Field:     "trip_orders.called_at",
				Direction: "desc",
			},
			MaxLimit:         50,
			MaxTimeRangeDays: 30,
		},
		RolePolicy: catalog.RolePolicy{
			ID:       "analyst",
			MaxLimit: 100,
		},
	}
}
