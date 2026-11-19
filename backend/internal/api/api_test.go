package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dylandlbv/Splitty/backend/internal/store"
)

func TestAuthFlowEndToEnd(t *testing.T) {
	st, err := store.Open(context.Background(), store.DriverSQLite, t.TempDir()+"/test.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	h := NewHandler(st, "test-secret", "")

	post := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	if w := post("/api/auth/register", `{"email":"a@b.c","password":"secret123","name":"A"}`); w.Code != http.StatusCreated && w.Code != http.StatusOK {
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
