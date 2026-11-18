package domain

import "sort"

type MemberBalance struct {
	UserID       string `json:"user_id"`
	PaidCents    int64  `json:"paid_cents"`
	ShareCents   int64  `json:"share_cents"`
	BalanceCents int64  `json:"balance_cents"`
}

type Transfer struct {
	FromUserID  string `json:"from_user_id"`
	ToUserID    string `json:"to_user_id"`
	AmountCents int64  `json:"amount_cents"`
}

type Settlement struct {
	Month            string          `json:"month"`
	TotalCommonCents int64           `json:"total_common_cents"`
	Balances         []MemberBalance `json:"balances"`
	Transfers        []Transfer      `json:"transfers"`
}

// ComputeSettlement computes monthly balances and suggested transfers for a
// household. Only common expenses take part in the settlement; expenses with
// is_common=false are personal and ignored. In proportionally split expenses,
// members who declared no income for the month are excluded from the split;
// if nobody declared income, the expense falls back to an equal split.
func ComputeSettlement(month string, members []string, incomes map[string]int64, expenses []Expense) Settlement {
	sortedMembers := append([]string(nil), members...)
	sort.Strings(sortedMembers)

	balances := make([]MemberBalance, 0, len(sortedMembers))
	debts := make(map[string]int64, len(sortedMembers))
	for _, m := range sortedMembers {
		b := MemberBalance{UserID: m}
		balances = append(balances, b)
		debts[m] = 0
	}

	totalCommon := int64(0)
	for _, e := range expenses {
		if !e.IsCommon {
			continue
		}
		totalCommon += e.AmountCents
		for i := range balances {
			if balances[i].UserID == e.PaidBy {
				balances[i].PaidCents += e.AmountCents
			}
		}
		for uid, share := range sharesFor(e, sortedMembers, incomes) {
			for i := range balances {
				if balances[i].UserID == uid {
					balances[i].ShareCents += share
				}
			}
		}
	}

	for i := range balances {
		balances[i].BalanceCents = balances[i].PaidCents - balances[i].ShareCents
		debts[balances[i].UserID] = balances[i].BalanceCents
	}

	var debtors, creditors []string
	for _, m := range sortedMembers {
		if debts[m] < 0 {
			debtors = append(debtors, m)
		} else if debts[m] > 0 {
			creditors = append(creditors, m)
		}
	}
	sort.SliceStable(debtors, func(i, j int) bool { return debts[debtors[i]] < debts[debtors[j]] })
	sort.SliceStable(creditors, func(i, j int) bool { return debts[creditors[i]] > debts[creditors[j]] })

	transfers := make([]Transfer, 0, len(debtors)+len(creditors))
	di, ci := 0, 0
	for di < len(debtors) && ci < len(creditors) {
		d, c := debtors[di], creditors[ci]
		amount := -debts[d]
		if debts[c] < amount {
			amount = debts[c]
		}
		transfers = append(transfers, Transfer{FromUserID: d, ToUserID: c, AmountCents: amount})
		debts[d] += amount
		debts[c] -= amount
		if debts[d] == 0 {
			di++
		}
		if debts[c] == 0 {
			ci++
		}
	}

	return Settlement{
		Month:            month,
		TotalCommonCents: totalCommon,
		Balances:         balances,
		Transfers:        transfers,
	}
}

func sharesFor(e Expense, members []string, incomes map[string]int64) map[string]int64 {
	if e.SplitMode == SplitProportional {
		totalIncome := int64(0)
		for _, m := range members {
			if incomes[m] > 0 {
				totalIncome += incomes[m]
			}
		}
		if totalIncome > 0 {
			return sharesProportional(e.AmountCents, members, incomes, totalIncome)
		}
	}
	return sharesEqual(e.AmountCents, members)
}

func sharesProportional(amount int64, members []string, incomes map[string]int64, totalIncome int64) map[string]int64 {
	eligible := make([]string, 0, len(members))
	for _, m := range members {
		if incomes[m] > 0 {
			eligible = append(eligible, m)
		}
	}
	sort.Strings(eligible)

	shares := make(map[string]int64, len(eligible))
	allocated := int64(0)
	for _, m := range eligible {
		shares[m] = amount * incomes[m] / totalIncome
		allocated += shares[m]
	}
	for _, m := range eligible {
		if allocated >= amount {
			break
		}
		shares[m]++
		allocated++
	}
	return shares
}

func sharesEqual(amount int64, members []string) map[string]int64 {
	shares := make(map[string]int64, len(members))
	if len(members) == 0 || amount < 0 {
		return shares
	}
	sorted := append([]string(nil), members...)
	sort.Strings(sorted)
	base := amount / int64(len(sorted))
	remainder := amount - base*int64(len(sorted))
	for i, m := range sorted {
		if int64(i) < remainder {
			shares[m] = base + 1
		} else {
			shares[m] = base
		}
	}
	return shares
}
