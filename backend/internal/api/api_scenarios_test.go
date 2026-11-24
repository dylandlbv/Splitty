package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/dylandlbv/Splitty/backend/internal/domain"
	"github.com/dylandlbv/Splitty/backend/internal/store/storetest"
)

type testClient struct {
	t *testing.T
	h http.Handler
}

func newTestClient(t *testing.T) *testClient {
	t.Helper()
	st := storetest.OpenSQLite(t)
	return &testClient{t: t, h: NewHandler(st, "test-secret", "")}
}

type apiResponse struct {
	code int
	body []byte
}

func (c *testClient) do(method, path, token, body string) apiResponse {
	c.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	c.h.ServeHTTP(w, req)
	return apiResponse{code: w.Code, body: w.Body.Bytes()}
}

func (c *testClient) decode(body []byte, v any) {
	c.t.Helper()
	if err := json.Unmarshal(body, v); err != nil {
		c.t.Fatalf("decode response %q: %v", body, err)
	}
}

func (c *testClient) register(email, password, name string) (token, userID string) {
	c.t.Helper()
	r := c.do(http.MethodPost, "/api/auth/register", "", `{"email":"`+email+`","password":"`+password+`","name":"`+name+`"}`)
	if r.code != http.StatusCreated {
		c.t.Fatalf("register %s: %d %s", email, r.code, r.body)
	}
	var resp authResponse
	c.decode(r.body, &resp)
	if resp.Token == "" || resp.User.ID == "" {
		c.t.Fatalf("register %s: empty token or user: %s", email, r.body)
	}
	return resp.Token, resp.User.ID
}

func (c *testClient) expectCode(got apiResponse, want int, what string) {
	c.t.Helper()
	if got.code != want {
		c.t.Fatalf("%s: got %d, want %d (body: %s)", what, got.code, want, got.body)
	}
}

func (c *testClient) createHousehold(token, name string) string {
	c.t.Helper()
	r := c.do(http.MethodPost, "/api/households", token, `{"name":"`+name+`"}`)
	c.expectCode(r, http.StatusCreated, "create household")
	var h domain.Household
	c.decode(r.body, &h)
	if h.ID == "" {
		c.t.Fatalf("create household: empty id: %s", r.body)
	}
	return h.ID
}

func TestAuthValidationAndGuards(t *testing.T) {
	c := newTestClient(t)

	c.expectCode(c.do(http.MethodPost, "/api/auth/register", "", `{not json`), http.StatusBadRequest, "register invalid JSON")
	c.expectCode(c.do(http.MethodPost, "/api/auth/register", "", `{"email":"no-at-sign","password":"secret123","name":"A"}`), http.StatusBadRequest, "register invalid email")
	c.expectCode(c.do(http.MethodPost, "/api/auth/register", "", `{"email":"a@b.c","password":"short","name":"A"}`), http.StatusBadRequest, "register short password")

	token, userID := c.register("Alice@Example.com", "secret123", "Alice")

	c.expectCode(c.do(http.MethodPost, "/api/auth/register", "", `{"email":"alice@example.com","password":"otherpass1","name":"Dup"}`), http.StatusConflict, "duplicate register (case-insensitive email)")
	c.expectCode(c.do(http.MethodPost, "/api/auth/login", "", `{"email":"alice@example.com","password":"wrong-pass"}`), http.StatusUnauthorized, "login wrong password")
	c.expectCode(c.do(http.MethodPost, "/api/auth/login", "", `{"email":"ghost@example.com","password":"secret123"}`), http.StatusUnauthorized, "login unknown email")

	r := c.do(http.MethodPost, "/api/auth/login", "", `{"email":"ALICE@example.com","password":"secret123"}`)
	c.expectCode(r, http.StatusOK, "login (case-insensitive email)")

	c.expectCode(c.do(http.MethodGet, "/api/me", "", ""), http.StatusUnauthorized, "me without token")
	c.expectCode(c.do(http.MethodGet, "/api/me", "garbage.token.here", ""), http.StatusUnauthorized, "me with invalid token")
	c.expectCode(c.do(http.MethodGet, "/api/households", "", ""), http.StatusUnauthorized, "households without token")

	r = c.do(http.MethodGet, "/api/me", token, "")
	c.expectCode(r, http.StatusOK, "me with token")
	var me domain.User
	c.decode(r.body, &me)
	if me.ID != userID || me.Email != "alice@example.com" {
		t.Fatalf("me = %+v, want id %s email alice@example.com", me, userID)
	}
}

func TestHouseholdLifecycleAndPermissions(t *testing.T) {
	c := newTestClient(t)
	aliceToken, _ := c.register("alice@example.com", "secret123", "Alice")
	bobToken, _ := c.register("bob@example.com", "secret123", "Bob")
	eveToken, _ := c.register("eve@example.com", "secret123", "Eve")

	c.expectCode(c.do(http.MethodPost, "/api/households", aliceToken, `{"name":"  "}`), http.StatusBadRequest, "create household empty name")
	c.expectCode(c.do(http.MethodPost, "/api/households", aliceToken, `{bad`), http.StatusBadRequest, "create household invalid JSON")

	hid := c.createHousehold(aliceToken, "Coloc")

	r := c.do(http.MethodGet, "/api/households", aliceToken, "")
	c.expectCode(r, http.StatusOK, "list households")
	var list []domain.Household
	c.decode(r.body, &list)
	if len(list) != 1 || list[0].ID != hid {
		t.Fatalf("households = %+v, want [%s]", list, hid)
	}

	r = c.do(http.MethodGet, "/api/households/"+hid, aliceToken, "")
	c.expectCode(r, http.StatusOK, "get household")
	var detail householdDetail
	c.decode(r.body, &detail)
	if len(detail.Members) != 1 || detail.Members[0].Role != "admin" {
		t.Fatalf("members = %+v, want single admin", detail.Members)
	}

	c.expectCode(c.do(http.MethodGet, "/api/households/"+hid, bobToken, ""), http.StatusNotFound, "get household as non-member")
	c.expectCode(c.do(http.MethodPost, "/api/households/"+hid+"/expenses", bobToken, `{"name":"X","amount_cents":100,"split_mode":"equal","month":"2025-11"}`), http.StatusNotFound, "create expense as non-member")

	c.expectCode(c.do(http.MethodPost, "/api/households/"+hid+"/members", aliceToken, `{"email":"ghost@example.com"}`), http.StatusNotFound, "invite unknown email")

	r = c.do(http.MethodPost, "/api/households/"+hid+"/members", aliceToken, `{"email":"BOB@example.com"}`)
	c.expectCode(r, http.StatusCreated, "invite bob (case-insensitive email)")
	var member domain.Member
	c.decode(r.body, &member)
	if member.Role != "member" {
		t.Fatalf("invited member role = %q, want member", member.Role)
	}

	c.expectCode(c.do(http.MethodPost, "/api/households/"+hid+"/members", bobToken, `{"email":"eve@example.com"}`), http.StatusForbidden, "invite as non-admin member")

	r = c.do(http.MethodPost, "/api/households/"+hid+"/members", aliceToken, `{"email":"bob@example.com"}`)
	c.expectCode(r, http.StatusCreated, "invite existing member is idempotent")

	r = c.do(http.MethodGet, "/api/households/"+hid, aliceToken, "")
	c.decode(r.body, &detail)
	if len(detail.Members) != 2 {
		t.Fatalf("members = %+v, want 2", detail.Members)
	}

	r = c.do(http.MethodGet, "/api/households", eveToken, "")
	c.expectCode(r, http.StatusOK, "list households as outsider")
	c.decode(r.body, &list)
	if len(list) != 0 {
		t.Fatalf("outsider households = %+v, want empty", list)
	}
}

func TestIncomesFlow(t *testing.T) {
	c := newTestClient(t)
	aliceToken, aliceID := c.register("alice@example.com", "secret123", "Alice")
	bobToken, bobID := c.register("bob@example.com", "secret123", "Bob")
	hid := c.createHousehold(aliceToken, "Coloc")
	c.do(http.MethodPost, "/api/households/"+hid+"/members", aliceToken, `{"email":"bob@example.com"}`)

	incomesPath := "/api/households/" + hid + "/incomes/"
	c.expectCode(c.do(http.MethodPut, incomesPath+"2025-13", aliceToken, `{"incomes":[]}`), http.StatusBadRequest, "set incomes bad month")
	c.expectCode(c.do(http.MethodPut, incomesPath+"2025-11", aliceToken, `{"incomes":[{"user_id":"`+aliceID+`","amount_cents":-1}]}`), http.StatusBadRequest, "set incomes negative amount")
	c.expectCode(c.do(http.MethodPut, incomesPath+"2025-11", aliceToken, `{"incomes":[{"user_id":"not-a-member","amount_cents":100}]}`), http.StatusBadRequest, "set incomes non-member")
	c.expectCode(c.do(http.MethodGet, "/api/households/"+hid+"/incomes?month=nope", aliceToken, ""), http.StatusBadRequest, "get incomes bad month param")
	c.expectCode(c.do(http.MethodPut, incomesPath+"2025-11", bobToken, `{"incomes":[{"user_id":"`+aliceID+`","amount_cents":1}]}`), http.StatusOK, "any member can set incomes")

	body := `{"incomes":[{"user_id":"` + aliceID + `","amount_cents":300000},{"user_id":"` + bobID + `","amount_cents":150000}]}`
	r := c.do(http.MethodPut, incomesPath+"2025-11", aliceToken, body)
	c.expectCode(r, http.StatusOK, "set incomes")

	r = c.do(http.MethodGet, "/api/households/"+hid+"/incomes?month=2025-11", aliceToken, "")
	c.expectCode(r, http.StatusOK, "get incomes")
	var got []domain.Income
	c.decode(r.body, &got)
	want := map[string]int64{aliceID: 300000, bobID: 150000}
	if len(got) != 2 {
		t.Fatalf("incomes = %+v, want 2 rows", got)
	}
	for _, inc := range got {
		if inc.Month != "2025-11" || want[inc.UserID] != inc.AmountCents {
			t.Fatalf("income = %+v, want month 2025-11 amount %d", inc, want[inc.UserID])
		}
	}

	r = c.do(http.MethodGet, "/api/households/"+hid+"/incomes?month=2025-12", aliceToken, "")
	c.expectCode(r, http.StatusOK, "get incomes other month")
	c.decode(r.body, &got)
	if len(got) != 0 {
		t.Fatalf("incomes for other month = %+v, want empty", got)
	}

	r = c.do(http.MethodPut, incomesPath+"2025-11", aliceToken, `{"incomes":[{"user_id":"`+aliceID+`","amount_cents":250000}]}`)
	c.expectCode(r, http.StatusOK, "upsert incomes")
	r = c.do(http.MethodGet, "/api/households/"+hid+"/incomes?month=2025-11", aliceToken, "")
	c.decode(r.body, &got)
	for _, inc := range got {
		if inc.UserID == aliceID && inc.AmountCents != 250000 {
			t.Fatalf("alice income after upsert = %d, want 250000", inc.AmountCents)
		}
	}
}

func TestExpensesFlow(t *testing.T) {
	c := newTestClient(t)
	aliceToken, _ := c.register("alice@example.com", "secret123", "Alice")
	bobToken, bobID := c.register("bob@example.com", "secret123", "Bob")
	eveToken, _ := c.register("eve@example.com", "secret123", "Eve")
	hid := c.createHousehold(aliceToken, "Coloc")
	c.do(http.MethodPost, "/api/households/"+hid+"/members", aliceToken, `{"email":"bob@example.com"}`)

	expensesPath := "/api/households/" + hid + "/expenses"
	c.expectCode(c.do(http.MethodPost, expensesPath, aliceToken, `{bad`), http.StatusBadRequest, "create expense invalid JSON")
	c.expectCode(c.do(http.MethodPost, expensesPath, aliceToken, `{"name":"","amount_cents":100,"split_mode":"equal","month":"2025-11"}`), http.StatusBadRequest, "create expense empty name")
	c.expectCode(c.do(http.MethodPost, expensesPath, aliceToken, `{"name":"X","amount_cents":0,"split_mode":"equal","month":"2025-11"}`), http.StatusBadRequest, "create expense zero amount")
	c.expectCode(c.do(http.MethodPost, expensesPath, aliceToken, `{"name":"X","amount_cents":-5,"split_mode":"equal","month":"2025-11"}`), http.StatusBadRequest, "create expense negative amount")
	c.expectCode(c.do(http.MethodPost, expensesPath, aliceToken, `{"name":"X","amount_cents":100,"split_mode":"weird","month":"2025-11"}`), http.StatusBadRequest, "create expense bad split_mode")
	c.expectCode(c.do(http.MethodPost, expensesPath, aliceToken, `{"name":"X","amount_cents":100,"split_mode":"equal","month":"2025-13"}`), http.StatusBadRequest, "create expense bad month")
	c.expectCode(c.do(http.MethodPost, expensesPath, aliceToken, `{"name":"X","amount_cents":100,"paid_by":"not-a-member","split_mode":"equal","month":"2025-11"}`), http.StatusBadRequest, "create expense paid_by non-member")
	c.expectCode(c.do(http.MethodPost, expensesPath, eveToken, `{"name":"X","amount_cents":100,"split_mode":"equal","month":"2025-11"}`), http.StatusNotFound, "create expense as non-member")
	c.expectCode(c.do(http.MethodGet, expensesPath, aliceToken, ""), http.StatusBadRequest, "list expenses without month")

	r := c.do(http.MethodPost, expensesPath, aliceToken, `{"name":"Courses","amount_cents":10001,"split_mode":"equal","month":"2025-11"}`)
	c.expectCode(r, http.StatusCreated, "create expense (paid_by defaults to caller)")
	var e1 domain.Expense
	c.decode(r.body, &e1)
	if e1.Name != "Courses" || e1.AmountCents != 10001 || !e1.IsCommon || e1.SplitMode != domain.SplitEqual || e1.Month != "2025-11" || e1.PaidBy == "" {
		t.Fatalf("expense = %+v", e1)
	}

	r = c.do(http.MethodPost, expensesPath, bobToken, `{"name":"Loyer","amount_cents":90000,"paid_by":"`+bobID+`","is_common":true,"split_mode":"proportional","month":"2025-11"}`)
	c.expectCode(r, http.StatusCreated, "create expense paid by bob")
	var e2 domain.Expense
	c.decode(r.body, &e2)

	r = c.do(http.MethodPost, expensesPath, aliceToken, `{"name":"Perso","amount_cents":5000,"is_common":false,"split_mode":"equal","month":"2025-11"}`)
	c.expectCode(r, http.StatusCreated, "create personal expense")
	var e3 domain.Expense
	c.decode(r.body, &e3)
	if e3.IsCommon {
		t.Fatalf("personal expense flagged common: %+v", e3)
	}

	r = c.do(http.MethodPost, expensesPath, aliceToken, `{"name":"Autres","amount_cents":700,"split_mode":"equal","month":"2025-12"}`)
	c.expectCode(r, http.StatusCreated, "create expense other month")

	r = c.do(http.MethodGet, expensesPath+"?month=2025-11", aliceToken, "")
	c.expectCode(r, http.StatusOK, "list expenses 2025-11")
	var list []domain.Expense
	c.decode(r.body, &list)
	if len(list) != 3 {
		t.Fatalf("expenses 2025-11 = %+v, want 3", list)
	}
	r = c.do(http.MethodGet, expensesPath+"?month=2025-12", aliceToken, "")
	c.decode(r.body, &list)
	if len(list) != 1 {
		t.Fatalf("expenses 2025-12 = %+v, want 1", list)
	}

	c.expectCode(c.do(http.MethodDelete, "/api/households/"+hid+"/expenses/"+e3.ID, eveToken, ""), http.StatusNotFound, "delete expense as non-member")
	c.expectCode(c.do(http.MethodDelete, "/api/households/"+hid+"/expenses/"+e3.ID, aliceToken, ""), http.StatusNoContent, "delete expense")
	c.expectCode(c.do(http.MethodDelete, "/api/households/"+hid+"/expenses/"+e3.ID, aliceToken, ""), http.StatusNotFound, "delete expense twice")
	r = c.do(http.MethodGet, expensesPath+"?month=2025-11", bobToken, "")
	c.decode(r.body, &list)
	if len(list) != 2 {
		t.Fatalf("expenses after delete = %+v, want 2", list)
	}
}

func TestBalancesEndToEnd(t *testing.T) {
	c := newTestClient(t)
	aliceToken, aliceID := c.register("alice@example.com", "secret123", "Alice")
	bobToken, bobID := c.register("bob@example.com", "secret123", "Bob")
	chloeToken, chloeID := c.register("chloe@example.com", "secret123", "Chloe")
	hid := c.createHousehold(aliceToken, "Coloc")
	c.do(http.MethodPost, "/api/households/"+hid+"/members", aliceToken, `{"email":"bob@example.com"}`)
	c.do(http.MethodPost, "/api/households/"+hid+"/members", aliceToken, `{"email":"chloe@example.com"}`)

	month := "2025-11"
	c.do(http.MethodPut, "/api/households/"+hid+"/incomes/"+month, aliceToken,
		`{"incomes":[{"user_id":"`+aliceID+`","amount_cents":300000},{"user_id":"`+bobID+`","amount_cents":150000}]}`)

	expensesPath := "/api/households/" + hid + "/expenses"
	c.do(http.MethodPost, expensesPath, aliceToken, `{"name":"Courses","amount_cents":10001,"split_mode":"equal","month":"`+month+`"}`)
	c.do(http.MethodPost, expensesPath, bobToken, `{"name":"Loyer","amount_cents":90000,"split_mode":"proportional","month":"`+month+`"}`)
	c.do(http.MethodPost, expensesPath, aliceToken, `{"name":"Perso","amount_cents":50000,"is_common":false,"split_mode":"proportional","month":"`+month+`"}`)
	c.do(http.MethodPost, expensesPath, chloeToken, `{"name":"Autre mois","amount_cents":999,"split_mode":"equal","month":"2025-12"}`)

	c.expectCode(c.do(http.MethodGet, "/api/households/"+hid+"/balances/2025-13", aliceToken, ""), http.StatusBadRequest, "balances bad month")

	r := c.do(http.MethodGet, "/api/households/"+hid+"/balances/"+month, aliceToken, "")
	c.expectCode(r, http.StatusOK, "balances")
	var got domain.Settlement
	c.decode(r.body, &got)

	if got.Month != month {
		t.Fatalf("settlement month = %q, want %q", got.Month, month)
	}
	if got.TotalCommonCents != 100001 {
		t.Fatalf("total common = %d, want 100001 (personal expense must be excluded)", got.TotalCommonCents)
	}

	r = c.do(http.MethodGet, expensesPath+"?month="+month, aliceToken, "")
	var expenses []domain.Expense
	c.decode(r.body, &expenses)
	r = c.do(http.MethodGet, "/api/households/"+hid+"/incomes?month="+month, aliceToken, "")
	var incomeRows []domain.Income
	c.decode(r.body, &incomeRows)
	incomes := map[string]int64{}
	for _, inc := range incomeRows {
		incomes[inc.UserID] = inc.AmountCents
	}
	members := []string{aliceID, bobID, chloeID}
	want := domain.ComputeSettlement(month, members, incomes, expenses)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("settlement = %+v\nwant %+v", got, want)
	}

	shareByUser := map[string]int64{}
	for _, b := range got.Balances {
		shareByUser[b.UserID] = b.ShareCents
	}
	if shareByUser[chloeID] <= 0 || shareByUser[chloeID] > 3334 {
		t.Fatalf("chloe share = %d, want only her equal share of the 10001 expense (<= 3334, proportional expense must exclude her)", shareByUser[chloeID])
	}

	var shareSum, transferSum, debtSum int64
	for _, b := range got.Balances {
		shareSum += b.ShareCents
		if b.BalanceCents < 0 {
			debtSum += -b.BalanceCents
		}
	}
	for _, tr := range got.Transfers {
		transferSum += tr.AmountCents
	}
	if shareSum != got.TotalCommonCents {
		t.Fatalf("shares sum = %d, want total common %d", shareSum, got.TotalCommonCents)
	}
	if transferSum != debtSum {
		t.Fatalf("transfers sum = %d, want sum of debts %d", transferSum, debtSum)
	}
	if len(got.Transfers) == 0 {
		t.Fatalf("transfers empty, want at least one")
	}
}
