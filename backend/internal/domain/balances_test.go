package domain

import "testing"

func TestComputeSettlementProportional(t *testing.T) {
	members := []string{"alice", "bob", "chloe"}
	incomes := map[string]int64{"alice": 200000, "bob": 100000}
	expenses := []Expense{
		{Name: "Loyer", AmountCents: 90000, PaidBy: "alice", IsCommon: true, SplitMode: SplitProportional},
	}
	got := ComputeSettlement("2025-11", members, incomes, expenses)

	for _, b := range got.Balances {
		switch b.UserID {
		case "alice":
			if b.PaidCents != 90000 || b.ShareCents != 60000 || b.BalanceCents != 30000 {
				t.Errorf("alice = %+v, want paid 90000 share 60000 balance 30000", b)
			}
		case "bob":
			if b.ShareCents != 30000 || b.BalanceCents != -30000 {
				t.Errorf("bob = %+v, want share 30000 balance -30000", b)
			}
		case "chloe":
			if b.ShareCents != 0 || b.BalanceCents != 0 {
				t.Errorf("chloe = %+v, want share 0 balance 0", b)
			}
		}
	}
	if len(got.Transfers) != 1 || got.Transfers[0].FromUserID != "bob" || got.Transfers[0].ToUserID != "alice" || got.Transfers[0].AmountCents != 30000 {
		t.Errorf("transfers = %+v, want bob -> alice 30000", got.Transfers)
	}
	if got.TotalCommonCents != 90000 {
		t.Errorf("total = %d, want 90000", got.TotalCommonCents)
	}
}

func TestComputeSettlementEqual(t *testing.T) {
	members := []string{"alice", "bob", "chloe"}
	expenses := []Expense{
		{Name: "Courses", AmountCents: 90000, PaidBy: "bob", IsCommon: true, SplitMode: SplitEqual},
	}
	got := ComputeSettlement("2025-11", members, map[string]int64{}, expenses)
	for _, b := range got.Balances {
		if b.ShareCents != 30000 {
			t.Errorf("%s share = %d, want 30000", b.UserID, b.ShareCents)
		}
	}
	if len(got.Transfers) != 2 {
		t.Fatalf("transfers = %+v, want 2 transfers", got.Transfers)
	}
	total := int64(0)
	for _, tr := range got.Transfers {
		total += tr.AmountCents
		if tr.ToUserID != "bob" {
			t.Errorf("transfer to %s, want bob", tr.ToUserID)
		}
	}
	if total != 60000 {
		t.Errorf("transferred %d, want 60000", total)
	}
}

func TestComputeSettlementProportionalFallsBackToEqual(t *testing.T) {
	members := []string{"alice", "bob"}
	expenses := []Expense{
		{Name: "Courses", AmountCents: 10001, PaidBy: "alice", IsCommon: true, SplitMode: SplitProportional},
	}
	got := ComputeSettlement("2025-11", members, map[string]int64{}, expenses)
	for _, b := range got.Balances {
		if b.ShareCents != 5000 && b.ShareCents != 5001 {
			t.Errorf("%s share = %d, want 5000 or 5001", b.UserID, b.ShareCents)
		}
	}
}

func TestComputeSettlementIgnoresPersonalExpenses(t *testing.T) {
	members := []string{"alice", "bob"}
	incomes := map[string]int64{"alice": 100000, "bob": 100000}
	expenses := []Expense{
		{Name: "Perso", AmountCents: 50000, PaidBy: "alice", IsCommon: false, SplitMode: SplitProportional},
	}
	got := ComputeSettlement("2025-11", members, incomes, expenses)
	if got.TotalCommonCents != 0 {
		t.Errorf("total common = %d, want 0", got.TotalCommonCents)
	}
	for _, b := range got.Balances {
		if b.PaidCents != 0 || b.ShareCents != 0 || b.BalanceCents != 0 {
			t.Errorf("%s = %+v, want all zeros", b.UserID, b)
		}
	}
	if len(got.Transfers) != 0 {
		t.Errorf("transfers = %+v, want none", got.Transfers)
	}
}

func TestSharesRounding(t *testing.T) {
	members := []string{"alice", "bob", "chloe"}
	incomes := map[string]int64{"alice": 100000, "bob": 1}
	expenses := []Expense{
		{Name: "X", AmountCents: 10001, PaidBy: "alice", IsCommon: true, SplitMode: SplitProportional},
	}
	got := ComputeSettlement("2025-11", members, incomes, expenses)
	sum := int64(0)
	for _, b := range got.Balances {
		sum += b.ShareCents
	}
	if sum != 10001 {
		t.Errorf("shares sum = %d, want 10001", sum)
	}
}
