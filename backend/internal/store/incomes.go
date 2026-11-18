package store

import (
	"context"
	"time"

	"github.com/dylandlbv/Splitty/backend/internal/domain"
)

func (s *Store) GetIncomes(ctx context.Context, householdID string, month time.Time) ([]domain.Income, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT user_id, month, amount_cents FROM incomes
		 WHERE household_id = $1 AND month = $2`,
		householdID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Income{}
	for rows.Next() {
		var inc domain.Income
		var m time.Time
		if err := rows.Scan(&inc.UserID, &m, &inc.AmountCents); err != nil {
			return nil, err
		}
		inc.Month = domain.FormatMonth(m)
		out = append(out, inc)
	}
	return out, rows.Err()
}

func (s *Store) UpsertIncomes(ctx context.Context, householdID string, month time.Time, incomes []domain.Income) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, inc := range incomes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO incomes (household_id, user_id, month, amount_cents)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (household_id, user_id, month)
			 DO UPDATE SET amount_cents = EXCLUDED.amount_cents`,
			householdID, inc.UserID, month, inc.AmountCents,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
