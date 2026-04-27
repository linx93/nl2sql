package mysql_test

import (
	"context"
	"testing"
	"time"

	"nl2sql/internal/executor"
	"nl2sql/tests/testsupport/mysqlbootstrap"
)

func TestBuildMySQLContainerRequestUsesPinnedLocalImage(t *testing.T) {
	req := mysqlbootstrap.BuildMySQLContainerRequest()

	if req.ContainerRequest.Image != mysqlbootstrap.MySQLImageTag {
		t.Fatalf("expected container image %q, got %q", mysqlbootstrap.MySQLImageTag, req.ContainerRequest.Image)
	}
	if req.ContainerRequest.Env["MYSQL_DATABASE"] != "ride_hailing" {
		t.Fatalf("expected MYSQL_DATABASE to be ride_hailing, got %q", req.ContainerRequest.Env["MYSQL_DATABASE"])
	}
	if !req.Started {
		t.Fatalf("expected generic container request to start immediately")
	}
}

func TestExecutorRunsReadonlyQueryAgainstMySQL(t *testing.T) {
	env := mysqlbootstrap.StartMySQLContainer(t)
	defer env.Terminate(t)

	err := mysqlbootstrap.Bootstrap(context.Background(), env.RootDB, time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	readonlyDB := env.OpenReadonlyDB(t)
	exec := executor.NewMySQLExecutor(readonlyDB)
	result, err := exec.Query(context.Background(), "ride_hailing_ro", "SELECT COUNT(*) AS value FROM trip_orders", nil)
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(result.Rows) != 1 || len(result.Rows[0]) != 1 {
		t.Fatalf("expected one row with one column, got %#v", result.Rows)
	}

	value, ok := result.Rows[0][0].(int64)
	if !ok || value == 0 {
		t.Fatalf("expected first value to be positive int64, got %#v", result.Rows[0][0])
	}
}

