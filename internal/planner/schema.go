package planner

import (
	"encoding/json"
	"errors"
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
