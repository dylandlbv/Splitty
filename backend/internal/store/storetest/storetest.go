package storetest

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/dylandlbv/Splitty/backend/internal/store"
)

func OpenSQLite(t testing.TB) *store.Store {
	t.Helper()
	st, err := store.Open(context.Background(), store.DriverSQLite, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return st
}

func OpenPostgres(t testing.TB) *store.Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL-backed tests")
	}
	ctx := context.Background()
	reset, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open reset connection: %v", err)
	}
	if _, err := reset.ExecContext(ctx, `DROP SCHEMA public CASCADE`); err != nil {
		reset.Close()
		t.Fatalf("reset schema: %v", err)
	}
	if _, err := reset.ExecContext(ctx, `CREATE SCHEMA public`); err != nil {
		reset.Close()
		t.Fatalf("recreate schema: %v", err)
	}
	reset.Close()

	st, err := store.Open(ctx, store.DriverPostgres, dsn)
	if err != nil {
		t.Fatalf("open postgres store: %v", err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return st
}
