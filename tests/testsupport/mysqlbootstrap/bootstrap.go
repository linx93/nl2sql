package mysqlbootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// MySQLImageTag 固定测试使用的 MySQL 镜像版本，避免环境漂移。
	MySQLImageTag = "mysql:8.0.21"

	rideHailingDatabase = "ride_hailing"
	readonlyUsername    = "readonly"
	readonlyPassword    = "readonlypass"
)

// Environment 封装 bootstrap 相关的根连接、只读连接和容器生命周期。
type Environment struct {
	RootDB      *sql.DB
	RootDSN     string
	ReadonlyDSN string
	ReadonlyDB  *sql.DB
	container   testcontainers.Container
}

// StartMySQLContainer 启动一个本地 MySQL 容器，并返回后续 bootstrap 所需的连接信息。
func StartMySQLContainer(t testing.TB) *Environment {
	t.Helper()
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, BuildMySQLContainerRequest())
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

	if err := waitForPing(rootDB); err != nil {
		rootDB.Close()
		_ = container.Terminate(ctx)
		t.Skipf("skip MySQL integration test because ping failed: %v", err)
	}

	return &Environment{
		RootDB:      rootDB,
		RootDSN:     rootDSN,
		ReadonlyDSN: fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", readonlyUsername, readonlyPassword, host, port.Port(), rideHailingDatabase),
		container:   container,
	}
}

// OpenReadonlyDB 打开并缓存只读连接，确保后续查询路径不依赖 root 账号。
func (e *Environment) OpenReadonlyDB(t testing.TB) *sql.DB {
	t.Helper()

	if e.ReadonlyDB != nil {
		return e.ReadonlyDB
	}

	db, err := sql.Open("mysql", e.ReadonlyDSN)
	if err != nil {
		t.Fatalf("sql.Open readonly returned error: %v", err)
	}
	if err := waitForPing(db); err != nil {
		_ = db.Close()
		t.Fatalf("readonly ping failed: %v", err)
	}

	e.ReadonlyDB = db
	return db
}

// Terminate 释放 bootstrap 打开的数据库连接和测试容器。
func (e *Environment) Terminate(t testing.TB) {
	t.Helper()

	if e.ReadonlyDB != nil {
		_ = e.ReadonlyDB.Close()
	}
	if e.RootDB != nil {
		_ = e.RootDB.Close()
	}
	if e.container != nil {
		_ = e.container.Terminate(context.Background())
	}
}

// BuildMySQLContainerRequest 构造固定版本、固定初始库名的 MySQL 容器请求。
func BuildMySQLContainerRequest() testcontainers.GenericContainerRequest {
	return testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        MySQLImageTag,
			ExposedPorts: []string{"3306/tcp"},
			Env: map[string]string{
				"MYSQL_ROOT_PASSWORD": "rootpass",
				"MYSQL_DATABASE":      rideHailingDatabase,
			},
			WaitingFor: wait.ForLog("port: 3306  MySQL Community Server - GPL").WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	}
}

// Bootstrap 创建演示库表、审计表、只读账号，并写入动态种子数据。
func Bootstrap(ctx context.Context, rootDB *sql.DB, now time.Time) error {
	if rootDB == nil {
		return fmt.Errorf("root db is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	if _, err := rootDB.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS ride_hailing CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	if err := applyMigrationFile(ctx, rootDB, "0001_create_nl2sql_query_log.sql"); err != nil {
		return err
	}
	if err := applyMigrationFile(ctx, rootDB, "0002_create_ride_hailing_demo_tables.sql"); err != nil {
		return err
	}
	if err := ensureReadonlyUser(ctx, rootDB); err != nil {
		return err
	}
	if err := seedDemoData(ctx, rootDB, now.UTC()); err != nil {
		return err
	}

	return nil
}

func applyMigrationFile(ctx context.Context, rootDB *sql.DB, filename string) error {
	raw, err := os.ReadFile(filepath.Join(repoRoot(), "db", "migrations", filename))
	if err != nil {
		return fmt.Errorf("read migration %s: %w", filename, err)
	}

	for _, statement := range extractStatements(string(raw)) {
		if _, err := rootDB.ExecContext(ctx, qualifyCreateTable(statement)); err != nil {
			return fmt.Errorf("apply migration %s: %w", filename, err)
		}
	}

	return nil
}

func extractStatements(raw string) []string {
	segments := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), ";")
	keywords := []string{
		"CREATE TABLE",
		"ALTER TABLE",
		"INSERT INTO",
		"UPDATE ",
		"DELETE FROM",
		"GRANT ",
		"REVOKE ",
		"DROP TABLE",
	}

	statements := make([]string, 0, len(segments))
	for _, segment := range segments {
		trimmed := strings.TrimSpace(segment)
		if trimmed == "" {
			continue
		}

		upper := strings.ToUpper(trimmed)
		for _, keyword := range keywords {
			index := strings.Index(upper, keyword)
			if index < 0 {
				continue
			}

			statement := strings.TrimSpace(trimmed[index:])
			if statement != "" {
				statements = append(statements, statement)
			}
			break
		}
	}

	return statements
}

func qualifyCreateTable(statement string) string {
	const withExists = "CREATE TABLE IF NOT EXISTS "
	const withoutExists = "CREATE TABLE "

	upper := strings.ToUpper(statement)
	switch {
	case strings.HasPrefix(upper, withExists):
		return statement[:len(withExists)] + rideHailingDatabase + "." + statement[len(withExists):]
	case strings.HasPrefix(upper, withoutExists):
		return statement[:len(withoutExists)] + rideHailingDatabase + "." + statement[len(withoutExists):]
	default:
		return statement
	}
}

func ensureReadonlyUser(ctx context.Context, rootDB *sql.DB) error {
	if _, err := rootDB.ExecContext(ctx, fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'", readonlyUsername, readonlyPassword)); err != nil {
		return fmt.Errorf("create readonly user: %w", err)
	}
	if _, err := rootDB.ExecContext(ctx, fmt.Sprintf("GRANT SELECT ON %s.* TO '%s'@'%%'", rideHailingDatabase, readonlyUsername)); err != nil {
		return fmt.Errorf("grant readonly privilege: %w", err)
	}

	return nil
}

func seedDemoData(ctx context.Context, rootDB *sql.DB, now time.Time) error {
	if _, err := rootDB.ExecContext(ctx, "DELETE FROM ride_hailing.trip_orders"); err != nil {
		return fmt.Errorf("clear trip_orders: %w", err)
	}
	if _, err := rootDB.ExecContext(ctx, "DELETE FROM ride_hailing.drivers"); err != nil {
		return fmt.Errorf("clear drivers: %w", err)
	}

	drivers := []driverSeed{
		{DriverID: "d001", DriverName: "张三"},
		{DriverID: "d002", DriverName: "李四"},
		{DriverID: "d003", DriverName: "王五"},
	}
	for _, driver := range drivers {
		if _, err := rootDB.ExecContext(ctx, "INSERT INTO ride_hailing.drivers (driver_id, driver_name) VALUES (?, ?)", driver.DriverID, driver.DriverName); err != nil {
			return fmt.Errorf("insert driver %s: %w", driver.DriverID, err)
		}
	}

	for _, order := range buildSeedOrders(now) {
		if _, err := rootDB.ExecContext(ctx, "INSERT INTO ride_hailing.trip_orders (order_id, city_code, service_type, order_status, called_at, finished_at, is_cancelled, driver_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			order.OrderID,
			order.CityCode,
			order.ServiceType,
			order.OrderStatus,
			order.CalledAt,
			order.FinishedAt,
			order.IsCancelled,
			order.DriverID,
		); err != nil {
			return fmt.Errorf("insert order %s: %w", order.OrderID, err)
		}
	}

	return nil
}

func buildSeedOrders(now time.Time) []orderSeed {
	type cityPlan struct {
		CityCode string
		DriverID string
		Statuses []string
	}

	plans := []cityPlan{
		{
			CityCode: "310000",
			DriverID: "d001",
			Statuses: []string{"cancelled", "waiting_pickup", "cancelled", "completed", "cancelled", "waiting_pickup", "completed"},
		},
		{
			CityCode: "110000",
			DriverID: "d002",
			Statuses: []string{"cancelled", "completed", "completed", "waiting_pickup", "completed", "cancelled", "completed"},
		},
		{
			CityCode: "440100",
			DriverID: "d003",
			Statuses: []string{"completed", "completed", "cancelled", "completed", "waiting_pickup", "completed", "completed"},
		},
	}

	orders := make([]orderSeed, 0, 32)
	for _, plan := range plans {
		for dayOffset, status := range plan.Statuses {
			calledAt := time.Date(now.Year(), now.Month(), now.Day(), 10+dayOffset%3, 0, 0, 0, time.UTC).AddDate(0, 0, -dayOffset)
			order := orderSeed{
				OrderID:     fmt.Sprintf("%s-%02d", plan.CityCode, dayOffset),
				CityCode:    plan.CityCode,
				ServiceType: "快车",
				OrderStatus: status,
				CalledAt:    calledAt,
				FinishedAt:  nil,
				IsCancelled: 0,
				DriverID:    plan.DriverID,
			}

			switch status {
			case "cancelled":
				order.IsCancelled = 1
				order.DriverID = ""
			case "completed":
				order.FinishedAt = calledAt.Add(30 * time.Minute)
			case "waiting_pickup":
				order.FinishedAt = nil
			}

			orders = append(orders, order)
		}
	}

	orders = append(orders,
		orderSeed{
			OrderID:     "310000-older-35",
			CityCode:    "310000",
			ServiceType: "快车",
			OrderStatus: "cancelled",
			CalledAt:    now.AddDate(0, 0, -35),
			FinishedAt:  nil,
			IsCancelled: 1,
			DriverID:    "",
		},
		orderSeed{
			OrderID:     "110000-older-45",
			CityCode:    "110000",
			ServiceType: "快车",
			OrderStatus: "completed",
			CalledAt:    now.AddDate(0, 0, -45),
			FinishedAt:  now.AddDate(0, 0, -45).Add(30 * time.Minute),
			IsCancelled: 0,
			DriverID:    "d002",
		},
	)

	return orders
}

func repoRoot() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve repo root: runtime.Caller failed")
	}

	root := filepath.Dir(currentFile)
	for i := 0; i < 3; i++ {
		root = filepath.Dir(root)
	}

	return root
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

type driverSeed struct {
	DriverID   string
	DriverName string
}

type orderSeed struct {
	OrderID     string
	CityCode    string
	ServiceType string
	OrderStatus string
	CalledAt    time.Time
	FinishedAt  any
	IsCancelled int
	DriverID    string
}

