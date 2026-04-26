package mysql_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"nl2sql/internal/executor"
)

const mysqlImageTag = "mysql:8.0.21"

func TestBuildMySQLContainerRequestUsesPinnedLocalImage(t *testing.T) {
	req := buildMySQLContainerRequest()

	if req.ContainerRequest.Image != mysqlImageTag {
		t.Fatalf("expected container image %q, got %q", mysqlImageTag, req.ContainerRequest.Image)
	}
	if req.ContainerRequest.Env["MYSQL_DATABASE"] != "ride_hailing" {
		t.Fatalf("expected MYSQL_DATABASE to be ride_hailing, got %q", req.ContainerRequest.Env["MYSQL_DATABASE"])
	}
	if !req.Started {
		t.Fatalf("expected generic container request to start immediately")
	}
}

func TestExecutorRunsReadonlyQueryAgainstMySQL(t *testing.T) {
	env := startMySQLContainer(t)
	defer env.Terminate(t)

	exec := executor.NewMySQLExecutor(env.DB)
	result, err := exec.Query(context.Background(), "ride_hailing_ro", "SELECT 1 AS value", nil)
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(result.Rows) != 1 || len(result.Rows[0]) != 1 {
		t.Fatalf("expected one row with one column, got %#v", result.Rows)
	}

	value, ok := result.Rows[0][0].(int64)
	if !ok || value != 1 {
		t.Fatalf("expected first value to be int64(1), got %#v", result.Rows[0][0])
	}
}

type mysqlEnv struct {
	DB        *sql.DB
	container testcontainers.Container
}

func (e mysqlEnv) Terminate(t *testing.T) {
	t.Helper()

	if e.DB != nil {
		_ = e.DB.Close()
	}
	if e.container != nil {
		_ = e.container.Terminate(context.Background())
	}
}

func startMySQLContainer(t *testing.T) mysqlEnv {
	t.Helper()
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, buildMySQLContainerRequest())
	if err != nil {
		t.Skipf("skip MySQL integration test because Docker is unavailable: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("container.Host returned error: %v", err)
	}
	port, err := container.MappedPort(ctx, "3306/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("container.MappedPort returned error: %v", err)
	}

	rootDSN := fmt.Sprintf("root:rootpass@tcp(%s:%s)/mysql?parseTime=true", host, port.Port())
	rootDB, err := sql.Open("mysql", rootDSN)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer rootDB.Close()

	if err := waitForPing(rootDB); err != nil {
		_ = container.Terminate(ctx)
		t.Skipf("skip MySQL integration test because ping failed: %v", err)
	}

	// 创建只读测试账号，确保真正执行查询时不使用写权限账号。
	if _, err := rootDB.ExecContext(ctx, "CREATE USER IF NOT EXISTS 'readonly'@'%' IDENTIFIED BY 'readonlypass'"); err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("create readonly user failed: %v", err)
	}

	readonlyDSN := fmt.Sprintf("readonly:readonlypass@tcp(%s:%s)/?parseTime=true", host, port.Port())
	db, err := sql.Open("mysql", readonlyDSN)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("sql.Open readonly returned error: %v", err)
	}

	if err := waitForPing(db); err != nil {
		db.Close()
		_ = container.Terminate(ctx)
		t.Fatalf("readonly ping failed: %v", err)
	}

	return mysqlEnv{
		DB:        db,
		container: container,
	}
}

func buildMySQLContainerRequest() testcontainers.GenericContainerRequest {
	return testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        mysqlImageTag,
			ExposedPorts: []string{"3306/tcp"},
			Env: map[string]string{
				"MYSQL_ROOT_PASSWORD": "rootpass",
				"MYSQL_DATABASE":      "ride_hailing",
			},
			WaitingFor: wait.ForLog("port: 3306  MySQL Community Server - GPL").WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	}
}

func waitForPing(db *sql.DB) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if err := db.Ping(); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return db.Ping()
}
