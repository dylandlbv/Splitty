package api

import (
	"net/http"

	"github.com/dylandlbv/Splitty/backend/internal/store"
)

type Server struct {
	store  *store.Store
	secret string
}

func NewHandler(store *store.Store, secret, allowedOrigin string) http.Handler {
	s := &Server{store: store, secret: secret}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)

	mux.HandleFunc("GET /api/me", s.auth(s.handleMe))

	mux.HandleFunc("GET /api/households", s.auth(s.handleListHouseholds))
	mux.HandleFunc("POST /api/households", s.auth(s.handleCreateHousehold))
	mux.HandleFunc("GET /api/households/{id}", s.auth(s.handleGetHousehold))
	mux.HandleFunc("POST /api/households/{id}/members", s.auth(s.handleAddMember))
	mux.HandleFunc("POST /api/households/{id}/members/managed", s.auth(s.handleAddManagedMember))

	mux.HandleFunc("GET /api/households/{id}/incomes", s.auth(s.handleGetIncomes))
	mux.HandleFunc("PUT /api/households/{id}/incomes/{month}", s.auth(s.handleSetIncomes))

	mux.HandleFunc("GET /api/households/{id}/expenses", s.auth(s.handleListExpenses))
	mux.HandleFunc("POST /api/households/{id}/expenses", s.auth(s.handleCreateExpense))
	mux.HandleFunc("DELETE /api/households/{id}/expenses/{expenseID}", s.auth(s.handleDeleteExpense))

	mux.HandleFunc("GET /api/households/{id}/balances/{month}", s.auth(s.handleGetBalances))

	return withCORS(allowedOrigin)(mux)
}

func withCORS(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
