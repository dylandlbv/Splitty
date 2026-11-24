package domain

import (
	"errors"
	"time"
)

type SplitMode string

const (
	SplitEqual        SplitMode = "equal"
	SplitProportional SplitMode = "proportional"
)

func (m SplitMode) Valid() bool {
	return m == SplitEqual || m == SplitProportional
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	// HasLogin is false for members created by an admin without an account:
	// they cannot authenticate and their Email is never exposed (empty).
	HasLogin bool `json:"has_login"`
}

type Household struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Member struct {
	User User   `json:"user"`
	Role string `json:"role"`
}

type Income struct {
	UserID      string `json:"user_id"`
	Month       string `json:"month"`
	AmountCents int64  `json:"amount_cents"`
}

type Expense struct {
	ID          string    `json:"id"`
	HouseholdID string    `json:"household_id"`
	Name        string    `json:"name"`
	AmountCents int64     `json:"amount_cents"`
	PaidBy      string    `json:"paid_by"`
	IsCommon    bool      `json:"is_common"`
	SplitMode   SplitMode `json:"split_mode"`
	Month       string    `json:"month"`
	CreatedAt   time.Time `json:"created_at"`
}

const monthLayout = "2006-01"

// ParseMonth parses a month in YYYY-MM format.
func ParseMonth(s string) (time.Time, error) {
	t, err := time.Parse(monthLayout, s)
	if err != nil {
		return time.Time{}, errors.New("month must be in YYYY-MM format")
	}
	return t, nil
}

func FormatMonth(t time.Time) string {
	return t.Format(monthLayout)
}
