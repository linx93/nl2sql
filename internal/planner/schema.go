package planner

import (
	"encoding/json"
	"errors"

	"nl2sql/internal/domain"
)

// ValidateRawPlanJSON 校验模型输出的 RawPlan JSON 是否满足最基本合约。
func ValidateRawPlanJSON(raw []byte) error {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}

	if _, ok := payload["query_mode"]; !ok {
		return errors.New("query_mode is required")
	}

	return nil
}

// DecodeRawPlanJSON 校验并反序列化模型输出的 RawPlan JSON。
func DecodeRawPlanJSON(raw []byte) (domain.RawPlan, error) {
	if err := ValidateRawPlanJSON(raw); err != nil {
		return domain.RawPlan{}, err
	}

	var plan domain.RawPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return domain.RawPlan{}, err
	}

	return plan, nil
}
