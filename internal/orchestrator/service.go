package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"nl2sql/internal/audit"
	"nl2sql/internal/builder"
	"nl2sql/internal/catalog"
	"nl2sql/internal/domain"
	"nl2sql/internal/formatter"
	"nl2sql/internal/guard"
	"nl2sql/internal/resolver"
	pkgclock "nl2sql/pkg/clock"
)

var (
	// ErrPermissionDenied 表示当前用户角色或查询模式不具备执行权限。
	ErrPermissionDenied = errors.New("permission denied")
	// ErrUnsupportedDomain 表示请求引用了不存在或未启用的业务域。
	ErrUnsupportedDomain = errors.New("unsupported domain")
)

// Planner 定义自然语言到 RawPlan 的规划能力。
type Planner interface {
	// Plan 根据自然语言问题和领域生成原始计划。
	Plan(ctx context.Context, query string, domainID string) (domain.RawPlan, error)
}

// Executor 定义参数化 SQL 的执行能力。
type Executor interface {
	// Query 在指定 datasource_id 下执行 SQL 并返回待格式化的查询结果。
	Query(ctx context.Context, datasourceID string, sql string, args []any) (formatter.QueryResult, error)
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
	// clock 提供统一的当前时间来源，便于解析默认时间窗。
	clock pkgclock.Clock
}

// NewService 创建一个带有显式依赖注入的编排服务实例。
func NewService(cat catalog.Catalog, planner Planner, executor Executor, auditor Auditor, clk pkgclock.Clock) Service {
	return Service{
		catalog:  cat,
		planner:  planner,
		executor: executor,
		auditor:  auditor,
		clock:    clk,
	}
}

// Run 执行一条完整的 NL2SQL 请求链路。
func (s Service) Run(ctx context.Context, req QueryRequest) (Response, error) {
	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
	scopedCatalog, err := scopeCatalogToDomain(s.catalog, req.Domain)
	if err != nil {
		_ = s.persistAudit(ctx, audit.Entry{
			RequestID:            requestID,
			UserID:               req.UserID,
			UserRole:             req.UserRole,
			Domain:               req.Domain,
			ExecutionStatus:      "failed",
			RejectionStage:       "request_validation",
			ErrorMessageInternal: err.Error(),
		})
		return Response{}, err
	}

	scopedService := s
	scopedService.catalog = scopedCatalog
	role, err := scopedService.resolveRole(req.UserRole)
	if err != nil {
		_ = s.persistAudit(ctx, audit.Entry{
			RequestID:            requestID,
			UserID:               req.UserID,
			UserRole:             req.UserRole,
			Domain:               req.Domain,
			ExecutionStatus:      "failed",
			RejectionStage:       "request_validation",
			ErrorMessageInternal: err.Error(),
		})
		return Response{}, err
	}

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

	resolved, err := scopedService.resolve(raw, role)
	if err != nil {
		err = normalizeServiceError(err)
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

	built, err := scopedService.build(resolved)
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

	validated, err := scopedService.validate(resolved, role, built)
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

	queryResult, err := s.executor.Query(ctx, resolved.DatasourceID, validated.SQL, validated.Args)
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
		return resolver.ResolveDetail(raw, s.catalog, role, s.runtimeClock())
	}

	return resolver.ResolveAggregate(raw, s.catalog, role, s.runtimeClock())
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

func (s Service) resolveRole(requestRole string) (catalog.RolePolicy, error) {
	if requestRole == "" {
		return catalog.RolePolicy{}, fmt.Errorf("%w: user_role is required", ErrPermissionDenied)
	}

	role, ok := s.catalog.Roles[requestRole]
	if !ok {
		return catalog.RolePolicy{}, fmt.Errorf("%w: unknown role %s", ErrPermissionDenied, requestRole)
	}

	return role, nil
}

func (s Service) persistAudit(ctx context.Context, entry audit.Entry) error {
	if s.auditor == nil {
		return nil
	}

	return s.auditor.Save(ctx, entry)
}

func (s Service) runtimeClock() pkgclock.Clock {
	if s.clock != nil {
		return s.clock
	}

	return fixedClock{}
}

func scopeCatalogToDomain(cat catalog.Catalog, domainID string) (catalog.Catalog, error) {
	domainSpec, ok := cat.Domains[domainID]
	if !ok || !domainSpec.Enabled {
		return catalog.Catalog{}, fmt.Errorf("%w: %s", ErrUnsupportedDomain, domainID)
	}

	scoped := catalog.Catalog{
		Schemas:         append([]catalog.SchemaSnapshot(nil), cat.Schemas...),
		Domains:         map[string]catalog.DomainSpec{domainID: domainSpec},
		Metrics:         make(map[string]catalog.MetricSpec),
		Dimensions:      make(map[string]catalog.DimensionSpec),
		DetailViews:     make(map[string]catalog.DetailViewSpec),
		Roles:           make(map[string]catalog.RolePolicy),
		AliasesByDomain: make(map[string]catalog.AliasSet),
		TablesByName:    cloneTableSpecMap(cat.TablesByName),
		ColumnsByTable:  cloneColumnSpecMap(cat.ColumnsByTable),
	}

	if aliases, ok := cat.AliasesByDomain[domainID]; ok {
		scoped.AliasesByDomain[domainID] = aliases
	}
	for id, metric := range cat.Metrics {
		if metric.DomainID == domainID {
			scoped.Metrics[id] = metric
		}
	}
	for id, dimension := range cat.Dimensions {
		if dimension.DomainID == domainID {
			scoped.Dimensions[id] = dimension
		}
	}
	for id, detailView := range cat.DetailViews {
		if detailView.DomainID == domainID {
			scoped.DetailViews[id] = detailView
		}
	}
	for id, role := range cat.Roles {
		if role.DomainID == domainID {
			scoped.Roles[id] = role
		}
	}

	return scoped, nil
}

func cloneTableSpecMap(src map[string]catalog.TableSpec) map[string]catalog.TableSpec {
	cloned := make(map[string]catalog.TableSpec, len(src))
	for key, value := range src {
		cloned[key] = value
	}

	return cloned
}

func cloneColumnSpecMap(src map[string]map[string]catalog.ColumnSpec) map[string]map[string]catalog.ColumnSpec {
	cloned := make(map[string]map[string]catalog.ColumnSpec, len(src))
	for table, columns := range src {
		columnClone := make(map[string]catalog.ColumnSpec, len(columns))
		for name, spec := range columns {
			columnClone[name] = spec
		}
		cloned[table] = columnClone
	}

	return cloned
}

func normalizeServiceError(err error) error {
	if errors.Is(err, ErrPermissionDenied) || errors.Is(err, ErrUnsupportedDomain) {
		return err
	}

	message := err.Error()
	if strings.Contains(message, "query mode not allowed") || strings.Contains(message, "permission denied") {
		return fmt.Errorf("%w: %s", ErrPermissionDenied, message)
	}

	return err
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Now()
}
