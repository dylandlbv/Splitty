package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dylandlbv/Splitty/backend/internal/api"
	"github.com/dylandlbv/Splitty/backend/internal/store"
)

func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET must be set")
	}
	origin := os.Getenv("ALLOWED_ORIGIN")
	addr := envOr("ADDR", ":8080")

	driver, dsn, err := databaseConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	st, err := store.Open(ctx, driver, dsn)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer st.Close()

	if err := st.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: api.NewHandler(st, secret, origin),
	}
	log.Printf("splitty listening on %s (%s database)", addr, driver)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// databaseConfig resolves the database backend. SQLite is the default; set
// DB_DRIVER=postgres (and DATABASE_URL) to use PostgreSQL instead.
func databaseConfig() (string, string, error) {
	driver := envOr("DB_DRIVER", store.DriverSQLite)
	switch driver {
	case store.DriverSQLite:
		return driver, envOr("SQLITE_PATH", "./data/splitty.db"), nil
	case store.DriverPostgres:
		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			dsn = "postgres://splitty:splitty@localhost:5432/splitty?sslmode=disable"
		}
		return driver, dsn, nil
	default:
		return "", "", fmt.Errorf("unsupported DB_DRIVER %q (want %q or %q)", driver, store.DriverSQLite, store.DriverPostgres)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
