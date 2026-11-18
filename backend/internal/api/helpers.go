package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dylandlbv/Splitty/backend/internal/auth"
	"github.com/dylandlbv/Splitty/backend/internal/domain"
)

type ctxKey string

const userKey ctxKey = "user"

type authResponse struct {
	Token string      `json:"token"`
	User  domain.User `json:"user"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode[T any](r *http.Request) (T, error) {
	var v T
	err := json.NewDecoder(r.Body).Decode(&v)
	return v, err
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			writeErr(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		userID, err := auth.ParseToken(s.secret, token)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		user, err := s.store.UserByID(r.Context(), userID)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), userKey, user)
		next(w, r.WithContext(ctx))
	}
}

func currentUser(r *http.Request) domain.User {
	return r.Context().Value(userKey).(domain.User)
}
