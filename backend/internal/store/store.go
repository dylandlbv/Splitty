package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

const (
	DriverSQLite   = "sqlite"
	DriverPostgres = "postgres"
)

type Store struct {
	db     *sql.DB
	driver string
}

// Open opens the database identified by driver ("sqlite" or "postgres") and
// dsn. For SQLite, dsn is a file path; for Postgres, a connection string.
func Open(ctx context.Context, driver, dsn string) (*Store, error) {
	var driverName string
	switch driver {
	case DriverSQLite:
		if dir := filepath.Dir(dsn); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create sqlite directory: %w", err)
			}
		}
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn += sep + "_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
		driverName = DriverSQLite
	case DriverPostgres:
		driverName = "pgx"
	default:
		return nil, fmt.Errorf("unsupported database driver %q (want %q or %q)", driver, DriverSQLite, DriverPostgres)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	if driver == DriverSQLite {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, driver: driver}, nil
}

func (s *Store) Close() {
	s.db.Close()
}

type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var placeholderRe = regexp.MustCompile(`\$\d+`)

var rebindCache sync.Map

func (s *Store) rebind(query string) string {
	if s.driver != DriverSQLite {
		return query
	}
	if v, ok := rebindCache.Load(query); ok {
		return v.(string)
	}
	rebound := placeholderRe.ReplaceAllString(query, "?")
	rebindCache.Store(query, rebound)
	return rebound
}

func (s *Store) exec(ctx context.Context, tx dbtx, query string, args ...any) (sql.Result, error) {
	return tx.ExecContext(ctx, s.rebind(query), args...)
}

func (s *Store) query(ctx context.Context, tx dbtx, query string, args ...any) (*sql.Rows, error) {
	return tx.QueryContext(ctx, s.rebind(query), args...)
}

func (s *Store) queryRow(ctx context.Context, tx dbtx, query string, args ...any) *sql.Row {
	return tx.QueryRowContext(ctx, s.rebind(query), args...)
}

func (s *Store) begin(ctx context.Context) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, nil)
}

func scanTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t.UTC(), nil
	case string:
		return parseDBTime(t)
	case []byte:
		return parseDBTime(string(t))
	default:
		return time.Time{}, fmt.Errorf("unsupported time value %T", v)
	}
}

var timeLayouts = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func parseDBTime(s string) (time.Time, error) {
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time value %q", s)
}

// monthArg adapts a month value to the driver: Postgres receives a time.Time
// for its date columns, SQLite receives an ISO date string.
func (s *Store) monthArg(t time.Time) any {
	if s.driver == DriverSQLite {
		return t.Format("2006-01-02")
	}
	return t
}
