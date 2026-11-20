package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
