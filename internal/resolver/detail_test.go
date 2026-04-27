package resolver

import (
	"testing"

	"nl2sql/internal/domain"
)

func TestResolveDetailMapsSubjectToDetailViewAndRequiresNarrowingFilter(t *testing.T) {
	raw := domain.RawPlan{
		QueryMode:     "detail_list",
		DetailSubject: "待接驾订单",
		TimeRange: domain.RawTimeRange{
			Type:  "relative",
			Value: "last_7_days",
		},
	}

	_, err := ResolveDetail(raw, loadCatalogFixture(t), loadRoleFixture(t), fixedClock())
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if want := "narrowing filter"; err != nil && err.Error() != "detail query requires narrowing filter" {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestResolveDetailCanonicalizesAllowedFilterFieldAndOperator(t *testing.T) {
	raw := domain.RawPlan{
		QueryMode:     "detail_list",
		DetailSubject: "待接驾订单",
		Filters: []domain.RawFilter{
			{
				Field:    "city_code",
				Operator: "eq",
				Value:    "310000",
			},
		},
		TimeRange: domain.RawTimeRange{
			Type:  "relative",
			Value: "last_7_days",
		},
	}

	plan, err := ResolveDetail(raw, loadCatalogFixture(t), loadRoleFixture(t), fixedClock())
	if err != nil {
		t.Fatalf("ResolveDetail returned error: %v", err)
	}
	if len(plan.Filters) != 2 {
		t.Fatalf("expected city filter plus preset waiting_pickup filter, got %#v", plan.Filters)
	}
	if plan.Filters[0].FieldID != "trip_orders.city_code" {
		t.Fatalf("expected canonical filter field, got %q", plan.Filters[0].FieldID)
	}
	if plan.Filters[0].Operator != "=" {
		t.Fatalf("expected canonical operator '=', got %q", plan.Filters[0].Operator)
	}
}

func TestResolveDetailMapsDimensionAliasFilterField(t *testing.T) {
	raw := domain.RawPlan{
		QueryMode:     "detail_list",
		DetailSubject: "待接驾订单",
		Filters: []domain.RawFilter{
			{
				Field:    "城市",
				Operator: "eq",
				Value:    "310000",
			},
		},
		TimeRange: domain.RawTimeRange{
			Type:  "relative",
			Value: "last_7_days",
		},
	}

	plan, err := ResolveDetail(raw, loadCatalogFixture(t), loadRoleFixture(t), fixedClock())
	if err != nil {
		t.Fatalf("ResolveDetail returned error: %v", err)
	}
	if len(plan.Filters) != 2 {
		t.Fatalf("expected dimension alias filter plus preset waiting_pickup filter, got %#v", plan.Filters)
	}
	if plan.Filters[0].FieldID != "trip_orders.city_code" {
		t.Fatalf("expected dimension alias to resolve to trip_orders.city_code, got %q", plan.Filters[0].FieldID)
	}
}

func TestResolveDetailMapsCityAliasFilterValue(t *testing.T) {
	raw := domain.RawPlan{
		QueryMode:     "detail_list",
		DetailSubject: "待接驾订单",
		Filters: []domain.RawFilter{
			{
				Field:    "城市",
				Operator: "eq",
				Value:    "上海",
			},
		},
		TimeRange: domain.RawTimeRange{
			Type:  "relative",
			Value: "last_7_days",
		},
	}

	plan, err := ResolveDetail(raw, loadCatalogFixture(t), loadRoleFixture(t), fixedClock())
	if err != nil {
		t.Fatalf("ResolveDetail returned error: %v", err)
	}
	if plan.Filters[0].Value != "310000" {
		t.Fatalf("expected city alias value to resolve to 310000, got %#v", plan.Filters[0].Value)
	}
}

func TestResolveDetailAppliesPresetWaitingPickupFilter(t *testing.T) {
	raw := domain.RawPlan{
		QueryMode:     "detail_list",
		DetailSubject: "待接驾订单",
		Filters: []domain.RawFilter{
			{
				Field:    "城市",
				Operator: "eq",
				Value:    "上海",
			},
		},
		TimeRange: domain.RawTimeRange{
			Type:  "relative",
			Value: "last_7_days",
		},
	}

	plan, err := ResolveDetail(raw, loadCatalogFixture(t), loadRoleFixture(t), fixedClock())
	if err != nil {
		t.Fatalf("ResolveDetail returned error: %v", err)
	}
	if len(plan.Filters) != 2 {
		t.Fatalf("expected city filter plus preset waiting_pickup filter, got %#v", plan.Filters)
	}
	if plan.Filters[1].FieldID != "trip_orders.order_status" {
		t.Fatalf("expected preset filter field trip_orders.order_status, got %q", plan.Filters[1].FieldID)
	}
	if plan.Filters[1].Value != "waiting_pickup" {
		t.Fatalf("expected preset waiting_pickup value, got %#v", plan.Filters[1].Value)
	}
}

func TestResolveDetailRejectsWhenOnlyPresetFilterIsProvided(t *testing.T) {
	raw := domain.RawPlan{
		QueryMode:     "detail_list",
		DetailSubject: "待接驾订单",
		Filters: []domain.RawFilter{
			{
				Field:    "order_status",
				Operator: "eq",
				Value:    "waiting_pickup",
			},
		},
		TimeRange: domain.RawTimeRange{
			Type:  "relative",
			Value: "last_7_days",
		},
	}

	_, err := ResolveDetail(raw, loadCatalogFixture(t), loadRoleFixture(t), fixedClock())
	if err == nil {
		t.Fatalf("expected narrowing filter validation error")
	}
	if err.Error() != "detail query requires narrowing filter" {
		t.Fatalf("expected narrowing filter rejection, got %q", err.Error())
	}
}

func TestResolveDetailRejectsUnsafeFilterOperator(t *testing.T) {
	raw := domain.RawPlan{
		QueryMode:     "detail_list",
		DetailSubject: "待接驾订单",
		Filters: []domain.RawFilter{
			{
				Field:    "city_code",
				Operator: "OR 1=1 --",
				Value:    "310000",
			},
		},
		TimeRange: domain.RawTimeRange{
			Type:  "relative",
			Value: "last_7_days",
		},
	}

	_, err := ResolveDetail(raw, loadCatalogFixture(t), loadRoleFixture(t), fixedClock())
	if err == nil {
		t.Fatalf("expected operator rejection")
	}
	if err.Error() != "filter operator not allowed: OR 1=1 --" {
		t.Fatalf("expected operator rejection, got %q", err.Error())
	}
}
