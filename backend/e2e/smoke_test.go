package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dylandlbv/Splitty/backend/internal/domain"
)

// TestSmokeFullJourney builds the real splitty binary, starts it against a
// temporary SQLite database, and plays a complete user journey over HTTP:
// register, login, household, invite, incomes, expenses, balances, delete.
func TestSmokeFullJourney(t *testing.T) {
	bin := buildBinary(t)
	base := startServer(t, bin)
	c := &smokeClient{base: base, http: &http.Client{Timeout: 5 * time.Second}}

	aliceToken, aliceID := c.register(t, "alice@example.com", "secret123", "Alice")
	bobToken, bobID := c.register(t, "bob@example.com", "secret123", "Bob")
	eveToken, _ := c.register(t, "eve@example.com", "secret123", "Eve")

	if code, body := c.do(t, http.MethodPost, "/api/auth/login", "", map[string]string{"email": "alice@example.com", "password": "nope-nope"}); code != http.StatusUnauthorized {
		t.Fatalf("login wrong password: %d %s", code, body)
	}
	if code, body := c.do(t, http.MethodPost, "/api/auth/login", "", map[string]string{"email": "alice@example.com", "password": "secret123"}); code != http.StatusOK {
		t.Fatalf("login: %d %s", code, body)
	}

	code, body := c.do(t, http.MethodPost, "/api/households", aliceToken, map[string]string{"name": "Coloc"})
	if code != http.StatusCreated {
		t.Fatalf("create household: %d %s", code, body)
	}
	var household domain.Household
	json.Unmarshal(body, &household)
	hid := household.ID

	if code, _ := c.do(t, http.MethodGet, "/api/households/"+hid, eveToken, nil); code != http.StatusNotFound {
		t.Fatalf("outsider reading household: got %d, want 404", code)
	}

	code, body = c.do(t, http.MethodPost, "/api/households/"+hid+"/members", aliceToken, map[string]string{"email": "bob@example.com"})
	if code != http.StatusCreated {
		t.Fatalf("invite bob: %d %s", code, body)
	}

	month := "2025-11"
	code, body = c.do(t, http.MethodPut, "/api/households/"+hid+"/incomes/"+month, aliceToken, map[string]any{
		"incomes": []map[string]any{
			{"user_id": aliceID, "amount_cents": 300000},
			{"user_id": bobID, "amount_cents": 150000},
		},
	})
	if code != http.StatusOK {
		t.Fatalf("set incomes: %d %s", code, body)
	}

	code, body = c.do(t, http.MethodPost, "/api/households/"+hid+"/expenses", aliceToken, map[string]any{
		"name": "Courses", "amount_cents": 10001, "split_mode": "equal", "is_common": true, "month": month,
	})
	if code != http.StatusCreated {
		t.Fatalf("create equal expense: %d %s", code, body)
	}
	var equalExpense domain.Expense
	json.Unmarshal(body, &equalExpense)

	code, body = c.do(t, http.MethodPost, "/api/households/"+hid+"/expenses", bobToken, map[string]any{
		"name": "Loyer", "amount_cents": 90000, "paid_by": bobID, "split_mode": "proportional", "is_common": true, "month": month,
	})
	if code != http.StatusCreated {
		t.Fatalf("create proportional expense: %d %s", code, body)
	}
	var propExpense domain.Expense
	json.Unmarshal(body, &propExpense)

	code, body = c.do(t, http.MethodPost, "/api/households/"+hid+"/expenses", aliceToken, map[string]any{
		"name": "Perso", "amount_cents": 50000, "is_common": false, "split_mode": "equal", "month": month,
	})
	if code != http.StatusCreated {
		t.Fatalf("create personal expense: %d %s", code, body)
	}
	var personalExpense domain.Expense
	json.Unmarshal(body, &personalExpense)

	code, body = c.do(t, http.MethodGet, "/api/households/"+hid+"/expenses?month="+month, aliceToken, nil)
	if code != http.StatusOK {
		t.Fatalf("list expenses: %d %s", code, body)
	}
	var expenses []domain.Expense
	json.Unmarshal(body, &expenses)
	if len(expenses) != 3 {
		t.Fatalf("expenses = %d rows, want 3", len(expenses))
	}

	code, body = c.do(t, http.MethodGet, "/api/households/"+hid+"/incomes?month="+month, aliceToken, nil)
	if code != http.StatusOK {
		t.Fatalf("get incomes: %d %s", code, body)
	}
	var incomeRows []domain.Income
	json.Unmarshal(body, &incomeRows)
	incomes := map[string]int64{}
	for _, inc := range incomeRows {
		incomes[inc.UserID] = inc.AmountCents
	}

	code, body = c.do(t, http.MethodGet, "/api/households/"+hid+"/balances/"+month, aliceToken, nil)
	if code != http.StatusOK {
		t.Fatalf("balances: %d %s", code, body)
	}
	var got domain.Settlement
	json.Unmarshal(body, &got)

	want := domain.ComputeSettlement(month, []string{aliceID, bobID}, incomes, expenses)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("settlement = %+v\nwant %+v", got, want)
	}
	if got.TotalCommonCents != 100001 {
		t.Fatalf("total common = %d, want 100001 (personal expense excluded)", got.TotalCommonCents)
	}
	var debtSum, transferSum int64
	for _, b := range got.Balances {
		if b.BalanceCents < 0 {
			debtSum += -b.BalanceCents
		}
	}
	for _, tr := range got.Transfers {
		transferSum += tr.AmountCents
	}
	if len(got.Transfers) == 0 || transferSum != debtSum {
		t.Fatalf("transfers %+v (sum %d), want sum %d covering all debts", got.Transfers, transferSum, debtSum)
	}

	code, _ = c.do(t, http.MethodDelete, "/api/households/"+hid+"/expenses/"+personalExpense.ID, aliceToken, nil)
	if code != http.StatusNoContent {
		t.Fatalf("delete expense: got %d, want 204", code)
	}
	code, _ = c.do(t, http.MethodDelete, "/api/households/"+hid+"/expenses/"+personalExpense.ID, aliceToken, nil)
	if code != http.StatusNotFound {
		t.Fatalf("delete expense twice: got %d, want 404", code)
	}
	code, body = c.do(t, http.MethodGet, "/api/households/"+hid+"/balances/"+month, bobToken, nil)
	if code != http.StatusOK {
		t.Fatalf("balances after delete: %d %s", code, body)
	}
	var after domain.Settlement
	json.Unmarshal(body, &after)
	if after.TotalCommonCents != 100001 {
		t.Fatalf("total common after deleting personal expense = %d, want unchanged 100001", after.TotalCommonCents)
	}
	code, body = c.do(t, http.MethodDelete, "/api/households/"+hid+"/expenses/"+equalExpense.ID, aliceToken, nil)
	if code != http.StatusNoContent {
		t.Fatalf("delete equal expense: got %d, want 204", code)
	}
	code, body = c.do(t, http.MethodGet, "/api/households/"+hid+"/balances/"+month, aliceToken, nil)
	json.Unmarshal(body, &after)
	if after.TotalCommonCents != 90000 {
		t.Fatalf("total common after delete = %d, want 90000", after.TotalCommonCents)
	}

	code, body = c.do(t, http.MethodGet, "/api/households", eveToken, nil)
	if code != http.StatusOK {
		t.Fatalf("outsider list households: %d %s", code, body)
	}
	var eveHouseholds []domain.Household
	json.Unmarshal(body, &eveHouseholds)
	if len(eveHouseholds) != 0 {
		t.Fatalf("outsider households = %+v, want empty", eveHouseholds)
	}

	code, body = c.do(t, http.MethodGet, "/api/me", aliceToken, nil)
	if code != http.StatusOK || !strings.Contains(string(body), "alice@example.com") {
		t.Fatalf("me: %d %s", code, body)
	}
}

// TestServerRequiresJWTSecret checks the binary refuses to start without
// JWT_SECRET instead of serving unauthenticated tokens.
func TestServerRequiresJWTSecret(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin)
	env := []string{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "JWT_SECRET=") {
			env = append(env, kv)
		}
	}
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("server started without JWT_SECRET, want failure; output: %s", out)
	}
	if !strings.Contains(string(out), "JWT_SECRET") {
		t.Fatalf("output %q does not mention JWT_SECRET", out)
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "splitty")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/splitty")
	cmd.Dir = ".."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/splitty: %v\n%s", err, out)
	}
	return bin
}

func startServer(t *testing.T, bin string) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	dbPath := filepath.Join(t.TempDir(), "smoke.db")
	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(),
		"JWT_SECRET=e2e-smoke-secret",
		"SQLITE_PATH="+dbPath,
		fmt.Sprintf("ADDR=127.0.0.1:%d", port),
	)
	var logs bytes.Buffer
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if t.Failed() {
			t.Logf("server logs:\n%s", logs.String())
		}
	})

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(15 * time.Second)
	for {
		resp, err := client.Get(base + "/api/me")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusUnauthorized {
				return base
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("server not ready on %s; logs:\n%s", base, logs.String())
		}
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			t.Fatalf("server exited early; logs:\n%s", logs.String())
		}
		time.Sleep(100 * time.Millisecond)
	}
}

type smokeClient struct {
	base string
	http *http.Client
}

func (c *smokeClient) do(t *testing.T, method, path, token string, body any) (int, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, data
}

func (c *smokeClient) register(t *testing.T, email, password, name string) (token, userID string) {
	t.Helper()
	code, body := c.do(t, http.MethodPost, "/api/auth/register", "", map[string]string{"email": email, "password": password, "name": name})
	if code != http.StatusCreated {
		t.Fatalf("register %s: %d %s", email, code, body)
	}
	var resp struct {
		Token string      `json:"token"`
		User  domain.User `json:"user"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode register: %v (%s)", err, body)
	}
	if resp.Token == "" || resp.User.ID == "" {
		t.Fatalf("register %s: empty token/user: %s", email, body)
	}
	return resp.Token, resp.User.ID
}
