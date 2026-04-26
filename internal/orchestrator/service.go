package orchestrator

import (
	"context"
	"fmt"
	"time"

	"nl2sql/internal/audit"
	"nl2sql/internal/builder"
	"nl2sql/internal/catalog"
	"nl2sql/internal/domain"
	"nl2sql/internal/formatter"
	"nl2sql/internal/guard"
	"nl2sql/internal/resolver"
)

// Planner 定义自然语言到 RawPlan 的规划能力。
type Planner interface {
	// Plan 根据自然语言问题和领域生成原始计划。
	Plan(ctx context.Context, query string, domainID string) (domain.RawPlan, error)
}

// Executor 定义参数化 SQL 的执行能力。
type Executor interface {
	// Query 执行 SQL 并返回待格式化的查询结果。
	Query(ctx context.Context, sql string, args []any) (formatter.QueryResult, error)
}

// Auditor 定义审计日志持久化能力。
type Auditor interface {
	// Save 持久化一次完整的审计记录。
	Save(ctx context.Context, entry audit.Entry) error
}

// QueryRequest 表示应用层接收的 NL2SQL 查询请求。
type QueryRequest struct {
	// Query 是用户输入的自然语言问题。
	Query string
	// Domain 是用户显式指定的业务领域。
	Domain string
	// UserID 是请求发起者。
	UserID string
	// UserRole 是本次请求使用的角色。
	UserRole string
}

// Meta 表示响应中的执行元信息。
type Meta struct {
	// QueryMode 是最终命中的查询模式。
	QueryMode string
	// ResultKind 是 aggregate/detail 等结果类型。
	ResultKind string
	// RowCount 是本次返回行数。
	RowCount int
	// Truncated 表示结果是否被截断。
	Truncated bool
}

// Response 表示应用层返回的统一查询响应。
type Response struct {
	// RequestID 是请求唯一标识。
	RequestID string
	// Data 是格式化后的结构化结果。
	Data formatter.ResponseData
	// Meta 是本次请求的执行元信息。
	Meta Meta
}

// Service 协调 planning、resolution、build、guard、execute、format 和 audit。
type Service struct {
	// catalog 保存运行时已加载的目录信息。
	catalog catalog.Catalog
	// planner 负责自然语言到 RawPlan 的转换。
	planner Planner
	// executor 负责执行参数化 SQL。
	executor Executor
	// auditor 负责持久化请求审计日志。
	auditor Auditor
}

// Run 执行一条完整的 NL2SQL 请求链路。
func (s Service) Run(ctx context.Context, req QueryRequest) (Response, error) {
	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
	role := s.resolveRole(req.UserRole)

	raw, err := s.planner.Plan(ctx, req.Query, req.Domain)
	if err != nil {
		_ = s.persistAudit(ctx, audit.Entry{
			RequestID:            requestID,
			UserID:               req.UserID,
			UserRole:             role.ID,
			Domain:               req.Domain,
			ExecutionStatus:      "failed",
			RejectionStage:       "planning",
			ErrorMessageInternal: err.Error(),
		})
		return Response{}, err
	}

	resolved, err := s.resolve(raw, role)
	if err != nil {
		_ = s.persistAudit(ctx, audit.Entry{
			RequestID:            requestID,
			UserID:               req.UserID,
			UserRole:             role.ID,
			Domain:               req.Domain,
			ExecutionStatus:      "failed",
			RejectionStage:       "resolution",
			ErrorMessageInternal: err.Error(),
		})
		return Response{}, err
	}

	built, err := s.build(resolved)
	if err != nil {
		_ = s.persistAudit(ctx, audit.Entry{
			RequestID:            requestID,
			UserID:               req.UserID,
			UserRole:             role.ID,
			Domain:               req.Domain,
			DatasourceID:         resolved.DatasourceID,
			ExecutionStatus:      "failed",
			RejectionStage:       "build",
			ErrorMessageInternal: err.Error(),
		})
		return Response{}, err
	}

	validated, err := s.validate(resolved, role, built)
	if err != nil {
		_ = s.persistAudit(ctx, audit.Entry{
			RequestID:            requestID,
			UserID:               req.UserID,
			UserRole:             role.ID,
			Domain:               req.Domain,
			DatasourceID:         resolved.DatasourceID,
			ExecutionStatus:      "failed",
			RejectionStage:       "guard",
			ErrorMessageInternal: err.Error(),
		})
		return Response{}, err
	}

	queryResult, err := s.executor.Query(ctx, validated.SQL, validated.Args)
	if err != nil {
		_ = s.persistAudit(ctx, audit.Entry{
			RequestID:            requestID,
			UserID:               req.UserID,
			UserRole:             role.ID,
			Domain:               req.Domain,
			DatasourceID:         resolved.DatasourceID,
			ExecutionStatus:      "failed",
			RejectionStage:       "execution",
			ErrorMessageInternal: err.Error(),
		})
		return Response{}, err
	}

	queryResult.QueryMode = resolved.QueryMode
	queryResult.Limit = resolved.Limit
	if queryResult.ResultKind == "" {
		if resolved.QueryMode == domain.QueryModeDetailList {
			queryResult.ResultKind = "detail"
		} else {
			queryResult.ResultKind = "aggregate"
		}
	}

	formatted := formatter.Format(queryResult)
	if err := s.persistAudit(ctx, audit.Entry{
		RequestID:            requestID,
		UserID:               req.UserID,
		UserRole:             role.ID,
		Domain:               req.Domain,
		DatasourceID:         resolved.DatasourceID,
		NaturalLanguageQuery: req.Query,
		QueryMode:            string(resolved.QueryMode),
		ResultKind:           formatted.ResultKind,
		ExecutionStatus:      "success",
		ResultRowCount:       len(queryResult.Rows),
	}); err != nil {
		return Response{}, err
	}

	return Response{
		RequestID: requestID,
		Data:      formatted,
		Meta: Meta{
			QueryMode:  string(resolved.QueryMode),
			ResultKind: formatted.ResultKind,
			RowCount:   formatted.RowCount,
			Truncated:  formatted.Truncated,
		},
	}, nil
}

func (s Service) resolve(raw domain.RawPlan, role catalog.RolePolicy) (domain.ResolvedPlan, error) {
	if raw.QueryMode == string(domain.QueryModeDetailList) {
		return resolver.ResolveDetail(raw, s.catalog, role, fixedClock{})
	}

	return resolver.ResolveAggregate(raw, s.catalog, role, fixedClock{})
}

func (s Service) build(plan domain.ResolvedPlan) (builder.BuildResult, error) {
	if plan.QueryMode == domain.QueryModeDetailList {
		return builder.BuildDetail(plan, s.catalog)
	}

	return builder.BuildAggregate(plan, s.catalog)
}

func (s Service) validate(plan domain.ResolvedPlan, role catalog.RolePolicy, built builder.BuildResult) (builder.BuildResult, error) {
	input := guard.GuardInput{
		Plan:        plan,
		BuildResult: built,
		RolePolicy:  role,
	}
	if plan.DetailViewID != "" {
		input.DetailView = s.catalog.DetailViews[plan.DetailViewID]
	}

	return guard.Validate(input)
}

func (s Service) resolveRole(requestRole string) catalog.RolePolicy {
	if requestRole != "" {
		if role, ok := s.catalog.Roles[requestRole]; ok {
			return role
		}
	}
	if role, ok := s.catalog.Roles["analyst"]; ok {
		return role
	}

	return catalog.RolePolicy{}
}

func (s Service) persistAudit(ctx context.Context, entry audit.Entry) error {
	if s.auditor == nil {
		return nil
	}

	return s.auditor.Save(ctx, entry)
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Now()
}
