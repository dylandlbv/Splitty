package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dylandlbv/Splitty/backend/internal/auth"
	"github.com/dylandlbv/Splitty/backend/internal/domain"
	"github.com/dylandlbv/Splitty/backend/internal/store"
)

type householdDetail struct {
	Household domain.Household `json:"household"`
	Members   []domain.Member  `json:"members"`
}

func (s *Server) memberOr404(w http.ResponseWriter, r *http.Request, householdID string) (string, bool) {
	role, err := s.store.MemberRole(r.Context(), householdID, currentUser(r).ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "household not found")
		} else {
			writeErr(w, http.StatusInternalServerError, "internal error")
		}
		return "", false
	}
	return role, true
}

func (s *Server) handleListHouseholds(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListHouseholds(r.Context(), currentUser(r).ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if list == nil {
		list = []domain.Household{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateHousehold(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Name string `json:"name"`
	}](r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	h, err := s.store.CreateHousehold(r.Context(), auth.NewID(), name, currentUser(r).ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, h)
}

func (s *Server) handleGetHousehold(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.memberOr404(w, r, id); !ok {
		return
	}
	h, err := s.store.GetHousehold(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "household not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	members, err := s.store.ListMembers(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if members == nil {
		members = []domain.Member{}
	}
	writeJSON(w, http.StatusOK, householdDetail{Household: h, Members: members})
}

func (s *Server) handleAddMember(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	role, ok := s.memberOr404(w, r, id)
	if !ok {
		return
	}
	if role != "admin" {
		writeErr(w, http.StatusForbidden, "only admins can add members")
		return
	}
	req, err := decode[struct {
		Email string `json:"email"`
	}](r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	member, err := s.store.AddMember(r.Context(), id, email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "no user with this email; ask them to create an account first")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, member)
}

// handleAddManagedMember lets a household admin create a member that has no
// login (just a name). Incomes, expenses and balances are then managed for
// that member exactly like for any other member of the household.
func (s *Server) handleAddManagedMember(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	role, ok := s.memberOr404(w, r, id)
	if !ok {
		return
	}
	if role != "admin" {
		writeErr(w, http.StatusForbidden, "only admins can add members")
		return
	}
	req, err := decode[struct {
		Name string `json:"name"`
	}](r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	member, err := s.store.AddManagedMember(r.Context(), id, auth.NewID(), name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "household not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, member)
}

/// ---------- Incomes ----------

func (s *Server) handleGetIncomes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.memberOr404(w, r, id); !ok {
		return
	}
	month, ok := monthParamOr400(w, r.URL.Query().Get("month"))
	if !ok {
		return
	}
	mt, err := domain.ParseMonth(month)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	incomes, err := s.store.GetIncomes(r.Context(), id, mt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, incomes)
}

func (s *Server) handleSetIncomes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.memberOr404(w, r, id); !ok {
		return
	}
	month, ok := monthParamOr400(w, r.PathValue("month"))
	if !ok {
		return
	}
	mt, err := domain.ParseMonth(month)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	req, err := decode[struct {
		Incomes []domain.Income `json:"incomes"`
	}](r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	members, err := s.store.ListMembers(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	memberIDs := make(map[string]bool, len(members))
	for _, m := range members {
		memberIDs[m.User.ID] = true
	}

	for _, inc := range req.Incomes {
		if inc.AmountCents < 0 {
			writeErr(w, http.StatusBadRequest, "amounts cannot be negative")
			return
		}
		if !memberIDs[inc.UserID] {
			writeErr(w, http.StatusBadRequest, "user is not a member of this household")
			return
		}
	}

	if err := s.store.UpsertIncomes(r.Context(), id, mt, req.Incomes); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	saved, err := s.store.GetIncomes(r.Context(), id, mt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

/// ---------- Expenses ----------

type expenseRequest struct {
	Name        string           `json:"name"`
	AmountCents int64            `json:"amount_cents"`
	PaidBy      string           `json:"paid_by"`
	IsCommon    *bool            `json:"is_common"`
	SplitMode   domain.SplitMode `json:"split_mode"`
	Month       string           `json:"month"`
}

func (s *Server) handleCreateExpense(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.memberOr404(w, r, id); !ok {
		return
	}
	req, err := decode[expenseRequest](r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.AmountCents <= 0 {
		writeErr(w, http.StatusBadRequest, "amount must be greater than zero")
		return
	}
	if _, ok := monthParamOr400(w, req.Month); !ok {
		return
	}
	mt, err := domain.ParseMonth(req.Month)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !req.SplitMode.Valid() {
		writeErr(w, http.StatusBadRequest, "split_mode must be 'equal' or 'proportional'")
		return
	}
	paidBy := req.PaidBy
	if paidBy == "" {
		paidBy = currentUser(r).ID
	}

	members, err := s.store.ListMembers(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	memberIDs := make(map[string]bool, len(members))
	for _, m := range members {
		memberIDs[m.User.ID] = true
	}
	if !memberIDs[paidBy] {
		writeErr(w, http.StatusBadRequest, "paid_by must be a member of this household")
		return
	}

	isCommon := true
	if req.IsCommon != nil {
		isCommon = *req.IsCommon
	}

	e, err := s.store.CreateExpense(r.Context(), domain.Expense{
		ID:          auth.NewID(),
		HouseholdID: id,
		Name:        name,
		AmountCents: req.AmountCents,
		PaidBy:      paidBy,
		IsCommon:    isCommon,
		SplitMode:   req.SplitMode,
		Month:       req.Month,
	}, mt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (s *Server) handleListExpenses(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.memberOr404(w, r, id); !ok {
		return
	}
	month, ok := monthParamOr400(w, r.URL.Query().Get("month"))
	if !ok {
		return
	}
	mt, err := domain.ParseMonth(month)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	expenses, err := s.store.ListExpenses(r.Context(), id, mt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, expenses)
}

func (s *Server) handleDeleteExpense(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.memberOr404(w, r, id); !ok {
		return
	}
	deleted, err := s.store.DeleteExpense(r.Context(), id, r.PathValue("expenseID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !deleted {
		writeErr(w, http.StatusNotFound, "expense not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

/// ---------- Balances ----------

func (s *Server) handleGetBalances(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.memberOr404(w, r, id); !ok {
		return
	}
	month, ok := monthParamOr400(w, r.PathValue("month"))
	if !ok {
		return
	}
	mt, err := domain.ParseMonth(month)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	members, err := s.store.ListMembers(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	ids := make([]string, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.User.ID)
	}

	incomeRows, err := s.store.GetIncomes(r.Context(), id, mt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	incomes := make(map[string]int64, len(incomeRows))
	for _, inc := range incomeRows {
		incomes[inc.UserID] = inc.AmountCents
	}

	expenses, err := s.store.ListExpenses(r.Context(), id, mt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	settlement := domain.ComputeSettlement(month, ids, incomes, expenses)
	writeJSON(w, http.StatusOK, settlement)
}

func monthParamOr400(w http.ResponseWriter, raw string) (string, bool) {
	if _, err := domain.ParseMonth(raw); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return "", false
	}
	return raw, true
}
