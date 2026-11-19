package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylandlbv/Splitty/backend/internal/domain"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(context.Background(), DriverSQLite, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return st
}

func month(t *testing.T, s string) time.Time {
	t.Helper()
	m, err := domain.ParseMonth(s)
	if err != nil {
		t.Fatalf("parse month %q: %v", s, err)
	}
	return m
}

func TestMigrateTwice(t *testing.T) {
	st := openTestStore(t)
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestUsers(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	if _, err := st.CreateUser(ctx, "u1", "a@example.com", "Alice", "hash1"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := st.CreateUser(ctx, "u2", "a@example.com", "Bob", "hash2"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("want ErrEmailTaken, got %v", err)
	}

	u, hash, err := st.UserByEmail(ctx, "a@example.com")
	if err != nil || u.ID != "u1" || hash != "hash1" {
		t.Fatalf("UserByEmail: %v %+v %q", err, u, hash)
	}
	if _, _, err := st.UserByEmail(ctx, "nope@example.com"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if u, err := st.UserByID(ctx, "u1"); err != nil || u.Name != "Alice" {
		t.Fatalf("UserByID: %v %+v", err, u)
	}
	if _, err := st.UserByID(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestHouseholdsAndMembers(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	for _, u := range []struct{ id, email string }{
		{"u1", "a@example.com"}, {"u2", "b@example.com"},
	} {
		if _, err := st.CreateUser(ctx, u.id, u.email, "", "hash"); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}

	h, err := st.CreateHousehold(ctx, "h1", "Home", "u1")
	if err != nil || h.Name != "Home" {
		t.Fatalf("CreateHousehold: %v %+v", err, h)
	}

	list, err := st.ListHouseholds(ctx, "u1")
	if err != nil || len(list) != 1 || list[0].ID != "h1" {
		t.Fatalf("ListHouseholds: %v %+v", err, list)
	}
	if list, err := st.ListHouseholds(ctx, "u2"); err != nil || len(list) != 0 {
		t.Fatalf("ListHouseholds for outsider: %v %+v", err, list)
	}

	if h, err := st.GetHousehold(ctx, "h1"); err != nil || h.Name != "Home" {
		t.Fatalf("GetHousehold: %v %+v", err, h)
	}
	if _, err := st.GetHousehold(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	if role, err := st.MemberRole(ctx, "h1", "u1"); err != nil || role != "admin" {
		t.Fatalf("MemberRole: %v %q", err, role)
	}
	if _, err := st.MemberRole(ctx, "h1", "u2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	member, err := st.AddMember(ctx, "h1", "b@example.com")
	if err != nil || member.Role != "member" || member.User.ID != "u2" {
		t.Fatalf("AddMember: %v %+v", err, member)
	}
	if _, err := st.AddMember(ctx, "h1", "nope@example.com"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	members, err := st.ListMembers(ctx, "h1")
	if err != nil || len(members) != 2 {
		t.Fatalf("ListMembers: %v %+v", err, members)
	}
}

func TestIncomes(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	m := month(t, "2025-03")

	if _, err := st.CreateUser(ctx, "u1", "a@example.com", "", "hash"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := st.CreateHousehold(ctx, "h1", "Home", "u1"); err != nil {
		t.Fatalf("create household: %v", err)
	}
	if err := st.UpsertIncomes(ctx, "h1", m, []domain.Income{{UserID: "u1", AmountCents: 200000}}); err != nil {
		t.Fatalf("UpsertIncomes: %v", err)
	}
	if err := st.UpsertIncomes(ctx, "h1", m, []domain.Income{{UserID: "u1", AmountCents: 250000}}); err != nil {
		t.Fatalf("UpsertIncomes again: %v", err)
	}

	incomes, err := st.GetIncomes(ctx, "h1", m)
	if err != nil || len(incomes) != 1 {
		t.Fatalf("GetIncomes: %v %+v", err, incomes)
	}
	if incomes[0].Month != "2025-03" || incomes[0].AmountCents != 250000 {
		t.Fatalf("unexpected income: %+v", incomes[0])
	}
	if incomes, err := st.GetIncomes(ctx, "h1", month(t, "2025-04")); err != nil || len(incomes) != 0 {
		t.Fatalf("GetIncomes other month: %v %+v", err, incomes)
	}
}

func TestExpenses(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	m := month(t, "2025-03")

	if _, err := st.CreateUser(ctx, "u1", "a@example.com", "", "hash"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := st.CreateHousehold(ctx, "h1", "Home", "u1"); err != nil {
		t.Fatalf("create household: %v", err)
	}

	e, err := st.CreateExpense(ctx, domain.Expense{
		ID:          "e1",
		HouseholdID: "h1",
		Name:        "Groceries",
		AmountCents: 12000,
		PaidBy:      "u1",
		IsCommon:    true,
		SplitMode:   domain.SplitEqual,
	}, m)
	if err != nil {
		t.Fatalf("CreateExpense: %v", err)
	}
	if e.Month != "2025-03" || e.CreatedAt.IsZero() {
		t.Fatalf("unexpected expense: %+v", e)
	}

	list, err := st.ListExpenses(ctx, "h1", m)
	if err != nil || len(list) != 1 || list[0].ID != "e1" || list[0].Month != "2025-03" {
		t.Fatalf("ListExpenses: %v %+v", err, list)
	}
	if list, err := st.ListExpenses(ctx, "h1", month(t, "2025-04")); err != nil || len(list) != 0 {
		t.Fatalf("ListExpenses other month: %v %+v", err, list)
	}

	deleted, err := st.DeleteExpense(ctx, "h1", "e1")
	if err != nil || !deleted {
		t.Fatalf("DeleteExpense: %v %v", err, deleted)
	}
	if deleted, err := st.DeleteExpense(ctx, "h1", "e1"); err != nil || deleted {
		t.Fatalf("DeleteExpense twice: %v %v", err, deleted)
	}
}
