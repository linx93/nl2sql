# Verification Checklist

## 默认门禁

默认验证门禁就是完整执行 `go test ./...`。这条命令会覆盖：

- 单元测试
- smoke 测试
- MySQL integration 测试
- MiniMax live planner 测试
- MiniMax + MySQL live 查询流测试

不要把 live 测试从默认门禁里拆出去，除非设计文档先被显式更新。

## 必要环境变量

- `MINIMAX_API_KEY`：默认 live 测试必填
- `MINIMAX_MODEL`：可选，未设置时默认 `MiniMax-M2.7`
- `MINIMAX_BASE_URL`：可选，未设置时默认 `https://api.minimaxi.com`
- `MYSQL_RIDE_HAILING_ROOT_DSN`：本地 root DSN，供 bootstrap、排查和手工验证使用
- `MYSQL_RIDE_HAILING_RO_DSN`：运行时查询必须使用的只读 DSN
- `MYSQL_NL2SQL_AUDIT_DSN`：审计库独立部署时使用；未独立拆分时可直接指向业务库

## 必要环境条件

- Docker 可用：integration / live 测试会启动本地 MySQL 容器
- MiniMax Token Plan 配额可用：默认 live 测试会真实调用 MiniMax
- 只读账号可连通：查询执行路径不能回退到 root 或写账号

## 必跑命令

- `go test ./...`
- `go run ./cmd/nl2sqlctl config validate`
- `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/check-encoding.ps1`

## 提交前要求

- 本地 git hook 应指向 `.githooks`
- `go test ./...` 必须通过
- `scripts/check-encoding.ps1` 必须通过
- 中文内容必须人工确认无乱码

## 配置维护补充

- Schema 快照校验：`go run ./cmd/nl2sqlctl schema pull --datasource ride_hailing_ro`
- 保守脚手架预览：`go run ./cmd/nl2sqlctl scaffold domain --domain ride_hailing --datasource ride_hailing_ro --tables trip_orders`
