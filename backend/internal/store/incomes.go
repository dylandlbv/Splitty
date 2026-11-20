package store

import (
	"context"
	"time"

	"github.com/dylandlbv/Splitty/backend/internal/domain"
)

func (s *Store) GetIncomes(ctx context.Context, householdID string, month time.Time) ([]domain.Income, error) {
	rows, err := s.query(ctx, s.db,
		`SELECT user_id, month, amount_cents FROM incomes
		 WHERE household_id = $1 AND month = $2`,
		householdID, s.monthArg(month))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Income{}
	for rows.Next() {
		var inc domain.Income
		var m any
		if err := rows.Scan(&inc.UserID, &m, &inc.AmountCents); err != nil {
			return nil, err
		}
		monthTime, err := scanTime(m)
		if err != nil {
			return nil, err
		}
		inc.Month = domain.FormatMonth(monthTime)
		out = append(out, inc)
	}
	return out, rows.Err()
}

func (s *Store) UpsertIncomes(ctx context.Context, householdID string, month time.Time, incomes []domain.Income) error {
	tx, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, inc := range incomes {
		if _, err := s.exec(ctx, tx,
			`INSERT INTO incomes (household_id, user_id, month, amount_cents)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (household_id, user_id, month)
			 DO UPDATE SET amount_cents = EXCLUDED.amount_cents`,
			householdID, inc.UserID, s.monthArg(month), inc.AmountCents,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
