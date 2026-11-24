package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dylandlbv/Splitty/backend/internal/domain"
	"github.com/dylandlbv/Splitty/backend/internal/store/storetest"
)

func TestAuthFlowEndToEnd(t *testing.T) {
	st := storetest.OpenSQLite(t)
	h := NewHandler(st, "test-secret", "")

	post := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	if w := post("/api/auth/register", `{"email":"a@b.c","password":"secret123","name":"A"}`); w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body)
	}
	if w := post("/api/auth/register", `{"email":"a@b.c","password":"secret123","name":"Dup"}`); w.Code != http.StatusConflict {
		t.Fatalf("duplicate register: %d %s", w.Code, w.Body)
	}
	w := post("/api/auth/login", `{"email":"a@b.c","password":"secret123"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body)
	}
	if w := post("/api/auth/login", `{"email":"a@b.c","password":"wrong"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("bad login: %d %s", w.Code, w.Body)
	}
}

func TestManagedMemberFlow(t *testing.T) {
	st := storetest.OpenSQLite(t)
	h := NewHandler(st, "test-secret", "")

	do := func(method, path, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	decode := func(t *testing.T, w *httptest.ResponseRecorder, v any) {
		t.Helper()
		if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
			t.Fatalf("decode response %q: %v", w.Body.String(), err)
		}
	}
	register := func(t *testing.T, email string) string {
		t.Helper()
		w := do(http.MethodPost, "/api/auth/register", "", `{"email":"`+email+`","password":"secret123","name":"`+email+`"}`)
		if w.Code != http.StatusCreated {
			t.Fatalf("register %s: %d %s", email, w.Code, w.Body)
		}
		var res struct {
			Token string `json:"token"`
		}
		decode(t, w, &res)
		return res.Token
	}

	admin := register(t, "admin@x.y")
	member := register(t, "member@x.y")

	w := do(http.MethodPost, "/api/households", admin, `{"name":"Home"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create household: %d %s", w.Code, w.Body)
	}
	var hh domain.Household
	decode(t, w, &hh)

	if w := do(http.MethodPost, "/api/households/"+hh.ID+"/members", admin, `{"email":"member@x.y"}`); w.Code != http.StatusCreated {
		t.Fatalf("invite member: %d %s", w.Code, w.Body)
	}

	if w := do(http.MethodPost, "/api/households/"+hh.ID+"/members/managed", "", `{"name":"Junior"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("managed member without token: %d %s", w.Code, w.Body)
	}
	if w := do(http.MethodPost, "/api/households/"+hh.ID+"/members/managed", member, `{"name":"Junior"}`); w.Code != http.StatusForbidden {
		t.Fatalf("managed member as non-admin: %d %s", w.Code, w.Body)
	}
	if w := do(http.MethodPost, "/api/households/"+hh.ID+"/members/managed", admin, `{"name":"  "}`); w.Code != http.StatusBadRequest {
		t.Fatalf("managed member without name: %d %s", w.Code, w.Body)
	}
	if w := do(http.MethodPost, "/api/households/nope/members/managed", admin, `{"name":"Junior"}`); w.Code != http.StatusNotFound {
		t.Fatalf("managed member in unknown household: %d %s", w.Code, w.Body)
	}

	w = do(http.MethodPost, "/api/households/"+hh.ID+"/members/managed", admin, `{"name":"Junior"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create managed member: %d %s", w.Code, w.Body)
	}
	var created domain.Member
	decode(t, w, &created)
	if created.Role != "member" || created.User.Name != "Junior" || created.User.HasLogin || created.User.Email != "" {
		t.Fatalf("unexpected managed member: %+v", created)
	}
	managedID := created.User.ID

	w = do(http.MethodPut, "/api/households/"+hh.ID+"/incomes/2025-03", admin,
		`{"incomes":[{"user_id":"`+managedID+`","month":"2025-03","amount_cents":50000}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("set income for managed member: %d %s", w.Code, w.Body)
	}

	w = do(http.MethodPost, "/api/households/"+hh.ID+"/expenses", member,
		`{"name":"Courses","amount_cents":9000,"paid_by":"`+managedID+`","is_common":true,"split_mode":"equal","month":"2025-03"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("expense paid by managed member: %d %s", w.Code, w.Body)
	}

	w = do(http.MethodGet, "/api/households/"+hh.ID+"/balances/2025-03", admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("balances: %d %s", w.Code, w.Body)
	}
	var settlement domain.Settlement
	decode(t, w, &settlement)
	found := false
	for _, b := range settlement.Balances {
		if b.UserID == managedID {
			found = true
			if b.PaidCents != 9000 || b.ShareCents != 3000 {
				t.Fatalf("unexpected balance for managed member: %+v", b)
			}
		}
	}
	if !found {
		t.Fatalf("managed member missing from balances: %+v", settlement.Balances)
	}
}
