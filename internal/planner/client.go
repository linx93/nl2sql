package planner

import (
	"context"
	"errors"

	"nl2sql/internal/domain"
)

// Client 定义自然语言到 RawPlan 的规划客户端能力。
type Client interface {
	// Plan 根据问题文本和领域生成 RawPlan。
	Plan(ctx context.Context, query string, domainID string) (domain.RawPlan, error)
}

// StaticClient 是一个用于测试和离线联调的静态规划客户端。
type StaticClient struct {
	// RawPlan 是每次调用返回的固定计划。
	RawPlan domain.RawPlan
}

// Plan 返回预先注入的固定 RawPlan。
func (c StaticClient) Plan(_ context.Context, _ string, _ string) (domain.RawPlan, error) {
	if c.RawPlan.QueryMode == "" {
		return domain.RawPlan{}, errors.New("raw plan is empty")
	}

	return c.RawPlan, nil
}
