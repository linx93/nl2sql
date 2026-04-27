package audit

import (
	"context"
	"database/sql"
	"time"
)

// Entry 表示一次 NL2SQL 请求的审计记录。
type Entry struct {
	// RequestID 是请求唯一标识。
	RequestID string
	// UserID 是发起请求的用户标识。
	UserID string
	// UserRole 是请求使用的角色。
	UserRole string
	// Domain 是请求命中的业务领域。
	Domain string
	// DatasourceID 是最终访问的数据源标识。
	DatasourceID string
	// NaturalLanguageQuery 是原始自然语言问题。
	NaturalLanguageQuery string
	// RawPlanJSON 保存模型输出的原始计划。
	RawPlanJSON string
	// ResolvedPlanJSON 保存后端解析后的计划。
	ResolvedPlanJSON string
	// BuiltSQL 保存 builder 输出的 SQL。
	BuiltSQL string
	// ValidatedSQL 保存 guard 通过后的 SQL。
	ValidatedSQL string
	// QueryMode 保存最终查询模式。
	QueryMode string
	// ResultKind 保存 aggregate/detail 等结果类型。
	ResultKind string
	// DetailViewID 保存明细视图标识。
	DetailViewID string
	// RejectionStage 保存拒绝阶段。
	RejectionStage string
	// ExecutionStatus 保存 success/failed 等执行状态。
	ExecutionStatus string
	// ErrorCode 保存稳定错误码。
	ErrorCode string
	// ErrorMessageInternal 保存内部错误详情。
	ErrorMessageInternal string
	// ResultColumnsJSON 保存结果列预览。
	ResultColumnsJSON string
	// ResultPreviewJSON 保存结果行预览。
	ResultPreviewJSON string
	// ResultRowCount 保存返回行数。
	ResultRowCount int
	// LatencyMS 保存端到端耗时。
	LatencyMS int
	// LLMModel 保存使用的模型名。
	LLMModel string
	// PromptVersion 保存提示词版本。
	PromptVersion string
	// SQLFingerprint 保存 SQL 指纹。
	SQLFingerprint string
	// CreatedAt 保存审计记录创建时间。
	CreatedAt time.Time
}

// Repository 负责持久化 NL2SQL 查询审计日志。
type Repository struct {
	// db 是系统审计库使用的数据库连接。
	db *sql.DB
}

// NewRepository 创建审计仓储。
func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

// Save 将审计记录写入数据库。
func (r Repository) Save(ctx context.Context, entry Entry) error {
	query, args := BuildInsert(entry)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// BuildInsert 构造写入审计日志表的参数化 SQL。
func BuildInsert(entry Entry) (string, []any) {
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	query := "INSERT INTO nl2sql_query_log (request_id, user_id, user_role, domain, datasource_id, natural_language_query, raw_plan_json, resolved_plan_json, built_sql, validated_sql, query_mode, result_kind, detail_view_id, rejection_stage, execution_status, error_code, error_message_internal, result_columns_json, result_preview_json, result_row_count, latency_ms, llm_model, prompt_version, sql_fingerprint, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

	args := []any{
		entry.RequestID,
		entry.UserID,
		entry.UserRole,
		entry.Domain,
		entry.DatasourceID,
		entry.NaturalLanguageQuery,
		nullableJSON(entry.RawPlanJSON),
		nullableJSON(entry.ResolvedPlanJSON),
		entry.BuiltSQL,
		entry.ValidatedSQL,
		entry.QueryMode,
		entry.ResultKind,
		entry.DetailViewID,
		entry.RejectionStage,
		entry.ExecutionStatus,
		entry.ErrorCode,
		entry.ErrorMessageInternal,
		nullableJSON(entry.ResultColumnsJSON),
		nullableJSON(entry.ResultPreviewJSON),
		entry.ResultRowCount,
		entry.LatencyMS,
		entry.LLMModel,
		entry.PromptVersion,
		entry.SQLFingerprint,
		entry.CreatedAt,
	}

	return query, args
}

func nullableJSON(raw string) any {
	if raw == "" {
		return nil
	}

	return raw
}
