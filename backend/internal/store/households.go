package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/dylandlbv/Splitty/backend/internal/domain"
)

func (s *Store) CreateHousehold(ctx context.Context, id, name, ownerID string) (domain.Household, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Household{}, err
	}
	defer tx.Rollback(ctx)

	var h domain.Household
	if err := tx.QueryRow(ctx,
		`INSERT INTO households (id, name) VALUES ($1, $2) RETURNING id, name`,
		id, name,
	).Scan(&h.ID, &h.Name); err != nil {
		return domain.Household{}, err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO memberships (household_id, user_id, role) VALUES ($1, $2, 'admin')`,
		id, ownerID,
	); err != nil {
		return domain.Household{}, err
	}
	return h, tx.Commit(ctx)
}

func (s *Store) ListHouseholds(ctx context.Context, userID string) ([]domain.Household, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT h.id, h.name FROM households h
		 JOIN memberships m ON m.household_id = h.id
		 WHERE m.user_id = $1
		 ORDER BY h.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Household
	for rows.Next() {
		var h domain.Household
		if err := rows.Scan(&h.ID, &h.Name); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Store) GetHousehold(ctx context.Context, id string) (domain.Household, error) {
	var h domain.Household
	err := s.pool.QueryRow(ctx, `SELECT id, name FROM households WHERE id = $1`, id).Scan(&h.ID, &h.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Household{}, ErrNotFound
		}
		return domain.Household{}, err
	}
	return h, nil
}

func (s *Store) ListMembers(ctx context.Context, householdID string) ([]domain.Member, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT u.id, u.email, u.name, ms.role FROM memberships ms
		 JOIN users u ON u.id = ms.user_id
		 WHERE ms.household_id = $1
		 ORDER BY u.created_at, u.id`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Member
	for rows.Next() {
		var m domain.Member
		if err := rows.Scan(&m.User.ID, &m.User.Email, &m.User.Name, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) MemberRole(ctx context.Context, householdID, userID string) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx,
		`SELECT role FROM memberships WHERE household_id = $1 AND user_id = $2`,
		householdID, userID,
	).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return role, nil
}

func (s *Store) AddMember(ctx context.Context, householdID, email string) (domain.Member, error) {
	var userID string
	err := s.pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Member{}, ErrNotFound
		}
		return domain.Member{}, err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO memberships (household_id, user_id, role) VALUES ($1, $2, 'member')
		 ON CONFLICT (household_id, user_id) DO NOTHING`,
		householdID, userID,
	)
	if err != nil {
		return domain.Member{}, err
	}
	user, err := s.UserByID(ctx, userID)
	if err != nil {
		return domain.Member{}, err
	}
	return domain.Member{User: user, Role: "member"}, nil
}
