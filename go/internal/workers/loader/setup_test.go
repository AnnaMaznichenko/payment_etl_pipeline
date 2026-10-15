package loader

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testConn clickhouse.Conn

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)

	container, conn := startClickHouse(ctx)
	testConn = conn

	code := m.Run()

	_ = container.Terminate(context.Background())
	cancel()
	os.Exit(code)
}

func startClickHouse(ctx context.Context) (testcontainers.Container, clickhouse.Conn) {
	req := testcontainers.ContainerRequest{
		Image:        "clickhouse/clickhouse-server:latest",
		ExposedPorts: []string{"9000/tcp", "8123/tcp"},
		Env: map[string]string{
			"CLICKHOUSE_USER":     "default",
			"CLICKHOUSE_PASSWORD": "clickhouse_password",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("9000/tcp"),
			wait.ForHTTP("/").WithPort("8123/tcp"),
		).WithDeadline(120 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("start clickhouse: %v", err)
	}

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "9000")

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", host, port.Port())},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "clickhouse_password",
		},
	})
	if err != nil {
		log.Fatalf("open clickhouse: %v", err)
	}

	if err := conn.Ping(ctx); err != nil {
		log.Fatalf("ping clickhouse: %v", err)
	}

	dsn := fmt.Sprintf("clickhouse://default:clickhouse_password@%s:%s/default", host, port.Port())
	sqlDB, err := sql.Open("clickhouse", dsn)
	if err != nil {
		log.Fatalf("open sql connection: %v", err)
	}
	defer sqlDB.Close()

	if err := goose.SetDialect("clickhouse"); err != nil {
		log.Fatalf("goose dialect: %v", err)
	}
	if err := goose.Up(sqlDB, "../../../migrations/clickhouse"); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	return container, conn
}

func truncatePayments(t *testing.T, conn clickhouse.Conn) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := conn.Exec(ctx, "TRUNCATE TABLE "+tableName); err != nil {
			log.Printf("truncate payments: %v", err)
		}
	})
}
