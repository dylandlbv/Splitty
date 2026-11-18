package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dylandlbv/Splitty/backend/internal/domain"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrEmailTaken  = errors.New("email already used")
	ErrNotAdmitted = errors.New("user is not a member of this household")
)

func (s *Store) CreateUser(ctx context.Context, id, email, name, passwordHash string) (domain.User, error) {
	var u domain.User
	err := s.pool.QueryRow(ctx,
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
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, password_hash FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, "", ErrNotFound
		}
		return domain.User{}, "", err
	}
	return u, hash, nil
}

func (s *Store) UserByID(ctx context.Context, id string) (domain.User, error) {
	var u domain.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, err
	}
	return u, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
