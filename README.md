# NL2SQL

内部 NL2SQL 后端服务，使用 Go 实现。

## 项目范围

- v1 仅提供后端 API
- v1 仅支持 MySQL
- 运行时可连接多个只读数据源
- 单次请求只能命中一个数据源
- 支持聚合查询与受控明细列表查询
- 不支持任意 SQL 透传
- 不支持跨数据源查询

## 交付约束

- 核心生产行为必须遵守 TDD
- LLM 只能生成 `RawPlan`，不能直接生成生产 SQL
- SQL 必须由后端根据受控语义配置生成
- 所有 SQL 都必须经过 guard 校验
- 查询执行必须使用只读 MySQL 凭据
- 成功与失败两条审计路径都必须落库

## 关键文档

- `docs/project-constraints.md`
- `docs/plans/2026-04-27-nl2sql-design.md`
- `docs/plans/2026-04-27-nl2sql-implementation.md`
- `docs/plans/2026-04-27-minimax-live-e2e-design.md`
- `docs/plans/verification-checklist.md`

## 目录概览

- `cmd/server`：HTTP 服务入口
- `cmd/nl2sqlctl`：内部配置、schema 与校验工具
- `configs`：数据源配置、schema 快照、领域语义配置
- `db/migrations`：MySQL 建表与迁移 SQL
- `internal`：核心应用与领域代码
- `pkg`：小型复用工具
- `tests`：smoke、integration、live 测试

## 默认联调环境

默认 `go test ./...` 会执行 live MiniMax 与 live MySQL 链路测试，因此本地环境需要提前准备：

- `MINIMAX_API_KEY`：MiniMax Token Plan 可用密钥
- `MINIMAX_MODEL`：可选，未设置时默认使用 `MiniMax-M2.7`
- `MINIMAX_BASE_URL`：可选，未设置时默认使用 `https://api.minimaxi.com`
- `MYSQL_RIDE_HAILING_ROOT_DSN`：本地 bootstrap 和人工排查时使用的 root DSN
- `MYSQL_RIDE_HAILING_RO_DSN`：运行时查询必须使用的只读 DSN
- `MYSQL_NL2SQL_AUDIT_DSN`：可选；若审计库独立部署时使用
- Docker 可用：live / integration 测试会拉起本地 MySQL 容器

说明：

- 代码中的默认 live 测试会自己启动 MySQL 容器并动态初始化演示数据
- 查询执行链路必须始终走只读 DSN
- 审计落库可以使用独立审计 DSN，未独立拆分时可直接连接业务库

## 验证命令

- 全量测试：`go test ./...`
- Smoke 测试：`go test ./tests/smoke -v`
- Live 查询流：`go test ./tests/live -run TestLiveQueryFlow -v`
- 配置校验：`go run ./cmd/nl2sqlctl config validate`
- 编码检查：`powershell -NoProfile -ExecutionPolicy Bypass -File scripts/check-encoding.ps1`

## Git Hook

- 版本化 pre-commit hook 位于 `.githooks/pre-commit`
- 本地安装命令：`git config core.hooksPath .githooks`
- hook 会执行 `go test ./...` 与 `scripts/check-encoding.ps1`
- 未经明确批准，不应绕过该 hook 提交
