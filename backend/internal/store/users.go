package store

import (
	"context"
	"database/sql"
	"errors"

	pgconn "github.com/jackc/pgx/v5/pgconn"
	moderncsqlite "modernc.org/sqlite"

	"github.com/dylandlbv/Splitty/backend/internal/domain"
)

// SQLITE_CONSTRAINT_UNIQUE (extended result code).
const sqliteConstraintUnique = 19 | (8 << 8)

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
		if isUniqueViolation(err) {
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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	var sqliteErr *moderncsqlite.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == sqliteConstraintUnique
}
