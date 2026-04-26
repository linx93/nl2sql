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
