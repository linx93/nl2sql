package orchestrator

import "errors"

// ErrInvalidQuery 表示用户问题无法在当前安全约束下形成可执行查询。
var ErrInvalidQuery = errors.New("invalid query")
