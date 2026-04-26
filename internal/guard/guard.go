package guard

import (
	"fmt"
	"strings"

	"nl2sql/internal/builder"
	"nl2sql/internal/catalog"
	"nl2sql/internal/domain"
)

// GuardInput 表示 guard 校验执行前 SQL 所需的上下文。
type GuardInput struct {
	// Plan 是已解析完成的规范化查询计划。
	Plan domain.ResolvedPlan
	// BuildResult 是 SQL builder 产出的参数化结果。
	BuildResult builder.BuildResult
	// DetailView 是明细查询对应的受控视图定义。
	DetailView catalog.DetailViewSpec
	// RolePolicy 是本次请求使用的角色边界。
	RolePolicy catalog.RolePolicy
}

// Validate 对 builder 输出的 SQL 做最终策略校验。
func Validate(input GuardInput) (builder.BuildResult, error) {
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(input.BuildResult.SQL)), "SELECT ") {
		return builder.BuildResult{}, fmt.Errorf("invalid sql: only SELECT statements are allowed")
	}

	if input.Plan.QueryMode == domain.QueryModeDetailList {
		if err := validateDetailColumns(input.DetailView, input.BuildResult.ReferencedCols); err != nil {
			return builder.BuildResult{}, err
		}
		if err := validateDetailLimit(input.DetailView, input.RolePolicy, input.BuildResult.Limit); err != nil {
			return builder.BuildResult{}, err
		}
		if err := validateDetailComplexity(input.DetailView, input.BuildResult); err != nil {
			return builder.BuildResult{}, err
		}
	}

	return input.BuildResult, nil
}

func validateDetailColumns(detailView catalog.DetailViewSpec, referencedCols []string) error {
	allowed := make(map[string]struct{}, len(detailView.AllowedSelectColumns)+len(detailView.AllowedFilterFields)+2)
	for _, column := range detailView.AllowedSelectColumns {
		allowed[column] = struct{}{}
	}
	for _, column := range detailView.AllowedFilterFields {
		allowed[column] = struct{}{}
	}
	if detailView.RequiredTimeField != "" {
		allowed[detailView.RequiredTimeField] = struct{}{}
	}
	if detailView.DefaultSort.Field != "" {
		allowed[detailView.DefaultSort.Field] = struct{}{}
	}

	for _, column := range referencedCols {
		if _, ok := allowed[column]; !ok {
			return fmt.Errorf("column not allowed: %s", column)
		}
	}

	return nil
}

func validateDetailLimit(detailView catalog.DetailViewSpec, role catalog.RolePolicy, limit int) error {
	maxLimit := role.MaxLimit
	if detailView.MaxLimit > 0 && (maxLimit == 0 || detailView.MaxLimit < maxLimit) {
		maxLimit = detailView.MaxLimit
	}
	if maxLimit > 0 && limit > maxLimit {
		return fmt.Errorf("limit exceeds max %d", maxLimit)
	}

	return nil
}

func validateDetailComplexity(detailView catalog.DetailViewSpec, result builder.BuildResult) error {
	if result.JoinCount > len(detailView.AllowedJoins) {
		return fmt.Errorf("join count exceeds allowed joins")
	}
	if detailView.MaxTimeRangeDays > 0 && result.TimeRangeDays > detailView.MaxTimeRangeDays {
		return fmt.Errorf("time range exceeds max %d days", detailView.MaxTimeRangeDays)
	}

	return nil
}
