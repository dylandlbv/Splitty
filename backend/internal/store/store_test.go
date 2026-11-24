package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dylandlbv/Splitty/backend/internal/domain"
	"github.com/dylandlbv/Splitty/backend/internal/store"
	"github.com/dylandlbv/Splitty/backend/internal/store/storetest"
)

func month(t *testing.T, s string) time.Time {
	t.Helper()
	m, err := domain.ParseMonth(s)
	if err != nil {
		t.Fatalf("parse month %q: %v", s, err)
	}
	return m
}

func TestMigrateTwice(t *testing.T) {
	runMigrateTwice(t, storetest.OpenSQLite(t))
}

func TestUsers(t *testing.T) {
	runUsers(t, storetest.OpenSQLite(t))
}

func TestHouseholdsAndMembers(t *testing.T) {
	runHouseholdsAndMembers(t, storetest.OpenSQLite(t))
}

func TestIncomes(t *testing.T) {
	runIncomes(t, storetest.OpenSQLite(t))
}

func TestExpenses(t *testing.T) {
	runExpenses(t, storetest.OpenSQLite(t))
}

func TestManagedMembers(t *testing.T) {
	runManagedMembers(t, storetest.OpenSQLite(t))
}

func TestPostgresSuite(t *testing.T) {
	runMigrateTwice(t, storetest.OpenPostgres(t))
	runUsers(t, storetest.OpenPostgres(t))
	runHouseholdsAndMembers(t, storetest.OpenPostgres(t))
	runIncomes(t, storetest.OpenPostgres(t))
	runExpenses(t, storetest.OpenPostgres(t))
	runManagedMembers(t, storetest.OpenPostgres(t))
}

func runMigrateTwice(t *testing.T, st *store.Store) {
	t.Helper()
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func runUsers(t *testing.T, st *store.Store) {
	t.Helper()
	ctx := context.Background()

	if _, err := st.CreateUser(ctx, "u1", "a@example.com", "Alice", "hash1"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := st.CreateUser(ctx, "u2", "a@example.com", "Bob", "hash2"); !errors.Is(err, store.ErrEmailTaken) {
		t.Fatalf("want ErrEmailTaken, got %v", err)
	}
	if _, err := st.CreateUser(ctx, "u1", "other@example.com", "Dup", "hash3"); err == nil || errors.Is(err, store.ErrEmailTaken) {
		t.Fatalf("duplicate id must not map to ErrEmailTaken, got %v", err)
	}

	u, hash, err := st.UserByEmail(ctx, "a@example.com")
	if err != nil || u.ID != "u1" || hash != "hash1" {
		t.Fatalf("UserByEmail: %v %+v %q", err, u, hash)
	}
	if _, _, err := st.UserByEmail(ctx, "nope@example.com"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if u, err := st.UserByID(ctx, "u1"); err != nil || u.Name != "Alice" {
		t.Fatalf("UserByID: %v %+v", err, u)
	}
	if _, err := st.UserByID(ctx, "nope"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func runHouseholdsAndMembers(t *testing.T, st *store.Store) {
	t.Helper()
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
	if _, err := st.GetHousehold(ctx, "nope"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	if role, err := st.MemberRole(ctx, "h1", "u1"); err != nil || role != "admin" {
		t.Fatalf("MemberRole: %v %q", err, role)
	}
	if _, err := st.MemberRole(ctx, "h1", "u2"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	member, err := st.AddMember(ctx, "h1", "b@example.com")
	if err != nil || member.Role != "member" || member.User.ID != "u2" {
		t.Fatalf("AddMember: %v %+v", err, member)
	}
	if _, err := st.AddMember(ctx, "h1", "nope@example.com"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	members, err := st.ListMembers(ctx, "h1")
	if err != nil || len(members) != 2 {
		t.Fatalf("ListMembers: %v %+v", err, members)
	}
}

func runIncomes(t *testing.T, st *store.Store) {
	t.Helper()
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

func runExpenses(t *testing.T, st *store.Store) {
	t.Helper()
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

func runManagedMembers(t *testing.T, st *store.Store) {
	t.Helper()
	ctx := context.Background()
	m := month(t, "2025-03")

	if _, err := st.CreateUser(ctx, "u1", "a@example.com", "Alice", "hash"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := st.CreateHousehold(ctx, "h1", "Home", "u1"); err != nil {
		t.Fatalf("create household: %v", err)
	}

	member, err := st.AddManagedMember(ctx, "h1", "m1", "Junior")
	if err != nil {
		t.Fatalf("AddManagedMember: %v", err)
	}
	if member.Role != "member" || member.User.ID != "m1" || member.User.Name != "Junior" {
		t.Fatalf("unexpected member: %+v", member)
	}
	if member.User.HasLogin || member.User.Email != "" {
		t.Fatalf("managed member must have no login and no exposed email: %+v", member.User)
	}

	if _, err := st.AddManagedMember(ctx, "no-such-household", "m2", "Ghost"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("AddManagedMember unknown household: want ErrNotFound, got %v", err)
	}
	if _, err := st.UserByID(ctx, "m2"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("rolled-back managed user must not exist: %v", err)
	}

	if role, err := st.MemberRole(ctx, "h1", "m1"); err != nil || role != "member" {
		t.Fatalf("MemberRole for managed member: %v %q", err, role)
	}

	u, err := st.UserByID(ctx, "m1")
	if err != nil || u.HasLogin || u.Email != "" || u.Name != "Junior" {
		t.Fatalf("UserByID managed member: %v %+v", err, u)
	}
	if _, _, err := st.UserByEmail(ctx, "no-login+m1@splitty.invalid"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("managed member must not be reachable by email login: %v", err)
	}

	members, err := st.ListMembers(ctx, "h1")
	if err != nil || len(members) != 2 {
		t.Fatalf("ListMembers: %v %+v", err, members)
	}
	var managed *domain.Member
	for i := range members {
		if members[i].User.ID == "m1" {
			managed = &members[i]
		}
	}
	if managed == nil || managed.User.HasLogin || managed.User.Email != "" {
		t.Fatalf("managed member in list: %+v", members)
	}

	if err := st.UpsertIncomes(ctx, "h1", m, []domain.Income{{UserID: "m1", AmountCents: 50000}}); err != nil {
		t.Fatalf("UpsertIncomes for managed member: %v", err)
	}
	incomes, err := st.GetIncomes(ctx, "h1", m)
	if err != nil || len(incomes) != 1 || incomes[0].UserID != "m1" || incomes[0].AmountCents != 50000 {
		t.Fatalf("GetIncomes: %v %+v", err, incomes)
	}

	if _, err := st.CreateExpense(ctx, domain.Expense{
		ID:          "e1",
		HouseholdID: "h1",
		Name:        "Courses",
		AmountCents: 9000,
		PaidBy:      "m1",
		IsCommon:    true,
		SplitMode:   domain.SplitEqual,
	}, m); err != nil {
		t.Fatalf("CreateExpense paid by managed member: %v", err)
	}
}
