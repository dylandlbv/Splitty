package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/dylandlbv/Splitty/backend/internal/api"
	"github.com/dylandlbv/Splitty/backend/internal/store"
)

func main() {
	dsn := envOr("DATABASE_URL", "postgres://splitty:splitty@localhost:5432/splitty?sslmode=disable")
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET must be set")
	}
	origin := os.Getenv("ALLOWED_ORIGIN")
	addr := envOr("ADDR", ":8080")

	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
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
	log.Printf("splitty listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
