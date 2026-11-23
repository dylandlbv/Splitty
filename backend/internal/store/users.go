package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
		`INSERT INTO users (id, email, name, password_hash, has_login) VALUES ($1, $2, $3, $4, true)
		 RETURNING id, email, name, has_login`,
		id, email, name, passwordHash,
	).Scan(&u.ID, &u.Email, &u.Name, &u.HasLogin)
	if err != nil {
		if isEmailTaken(err) {
			return domain.User{}, ErrEmailTaken
		}
		return domain.User{}, err
	}
	return u, nil
}

// CreateManagedUser inserts a user row without any login credential: a
// synthetic unique email (never exposed by the API) and an empty password
// hash, so authentication against this row always fails.
func (s *Store) CreateManagedUser(ctx context.Context, tx dbtx, id, name string) (domain.User, error) {
	var u domain.User
	err := s.queryRow(ctx, tx,
		`INSERT INTO users (id, email, name, password_hash, has_login) VALUES ($1, $2, $3, '', false)
		 RETURNING id, email, name, has_login`,
		id, managedUserEmail(id), name,
	).Scan(&u.ID, &u.Email, &u.Name, &u.HasLogin)
	if err != nil {
		return domain.User{}, err
	}
	return publicUser(u), nil
}

// managedUserEmail builds the internal, unique placeholder email stored for
// members without a login. It is never returned by the API.
func managedUserEmail(id string) string {
	return fmt.Sprintf("no-login+%s@splitty.invalid", id)
}

// publicUser hides the synthetic email of members without a login.
func publicUser(u domain.User) domain.User {
	if !u.HasLogin {
		u.Email = ""
	}
	return u
}

func (s *Store) UserByEmail(ctx context.Context, email string) (domain.User, string, error) {
	var u domain.User
	var hash string
	err := s.queryRow(ctx, s.db,
		`SELECT id, email, name, password_hash, has_login FROM users WHERE email = $1 AND has_login`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &hash, &u.HasLogin)
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
		`SELECT id, email, name, has_login FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.HasLogin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, err
	}
	return publicUser(u), nil
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
