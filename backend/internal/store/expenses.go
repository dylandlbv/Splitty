package store

import (
	"context"
	"time"

	"github.com/dylandlbv/Splitty/backend/internal/domain"
)

func (s *Store) CreateExpense(ctx context.Context, e domain.Expense, month time.Time) (domain.Expense, error) {
	var createdAtVal, monthVal any
	err := s.queryRow(ctx, s.db,
		`INSERT INTO expenses (id, household_id, name, amount_cents, paid_by, is_common, split_mode, month)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at, month`,
		e.ID, e.HouseholdID, e.Name, e.AmountCents, e.PaidBy, e.IsCommon, e.SplitMode, s.monthArg(month),
	).Scan(&createdAtVal, &monthVal)
	if err != nil {
		return domain.Expense{}, err
	}
	createdAtTime, err := scanTime(createdAtVal)
	if err != nil {
		return domain.Expense{}, err
	}
	monthTime, err := scanTime(monthVal)
	if err != nil {
		return domain.Expense{}, err
	}
	e.CreatedAt = createdAtTime
	e.Month = domain.FormatMonth(monthTime)
	return e, nil
}

func (s *Store) ListExpenses(ctx context.Context, householdID string, month time.Time) ([]domain.Expense, error) {
	rows, err := s.query(ctx, s.db,
		`SELECT id, household_id, name, amount_cents, paid_by, is_common, split_mode, month, created_at
		 FROM expenses WHERE household_id = $1 AND month = $2
		 ORDER BY created_at, id`,
		householdID, s.monthArg(month))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Expense{}
	for rows.Next() {
		var e domain.Expense
		var createdAtVal, monthVal any
		if err := rows.Scan(&e.ID, &e.HouseholdID, &e.Name, &e.AmountCents, &e.PaidBy, &e.IsCommon, &e.SplitMode, &monthVal, &createdAtVal); err != nil {
			return nil, err
		}
		createdAtTime, err := scanTime(createdAtVal)
		if err != nil {
			return nil, err
		}
		monthTime, err := scanTime(monthVal)
		if err != nil {
			return nil, err
		}
		e.CreatedAt = createdAtTime
		e.Month = domain.FormatMonth(monthTime)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) DeleteExpense(ctx context.Context, householdID, id string) (bool, error) {
	tag, err := s.exec(ctx, s.db,
		`DELETE FROM expenses WHERE id = $1 AND household_id = $2`, id, householdID)
	if err != nil {
		return false, err
	}
	n, err := tag.RowsAffected()
	return n > 0, err
}
