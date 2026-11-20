package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	pgconn "github.com/jackc/pgx/v5/pgconn"
	moderncsqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/dylandlbv/Splitty/backend/internal/domain"
)

const (
	pgCodeUniqueViolation     = "23505"
	pgCodeForeignKeyViolation = "23503"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrEmailTaken  = errors.New("email already used")
	ErrNotAdmitted = errors.New("user is not a member of this household")
)

func (s *Store) CreateUser(ctx context.Context, id, email, name, passwordHash string) (domain.User, error) {
	var u domain.User
	err := s.queryRow(ctx, s.db,
		`INSERT INTO users (id, email, name, password_hash) VALUES ($1, $2, $3, $4)
		 RETURNING id, email, name`,
		id, email, name, passwordHash,
	).Scan(&u.ID, &u.Email, &u.Name)
	if err != nil {
		if isEmailTaken(err) {
			return domain.User{}, ErrEmailTaken
		}
		return domain.User{}, err
	}
	return u, nil
}

func (s *Store) UserByEmail(ctx context.Context, email string) (domain.User, string, error) {
	var u domain.User
	var hash string
	err := s.queryRow(ctx, s.db,
		`SELECT id, email, name, password_hash FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, "", ErrNotFound
		}
		return domain.User{}, "", err
	}
	return u, hash, nil
}

func (s *Store) UserByID(ctx context.Context, id string) (domain.User, error) {
	var u domain.User
	err := s.queryRow(ctx, s.db,
		`SELECT id, email, name FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, err
	}
	return u, nil
}

func isEmailTaken(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgCodeUniqueViolation {
		return pgErr.ConstraintName == "users_email_key"
	}
	var sqliteErr *moderncsqlite.Error
	if errors.As(err, &sqliteErr) &&
		(sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE || sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY) {
		return strings.Contains(sqliteErr.Error(), "users.email")
	}
	return false
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgCodeForeignKeyViolation
	}
	var sqliteErr *moderncsqlite.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY
}
