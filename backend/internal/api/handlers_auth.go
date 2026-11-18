package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dylandlbv/Splitty/backend/internal/auth"
	"github.com/dylandlbv/Splitty/backend/internal/store"
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	req, err := decode[credentials](r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !strings.Contains(email, "@") {
		writeErr(w, http.StatusBadRequest, "invalid email")
		return
	}
	if len(req.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	user, err := s.store.CreateUser(r.Context(), auth.NewID(), email, strings.TrimSpace(req.Name), hash)
	if err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			writeErr(w, http.StatusConflict, "email already used")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	token, err := auth.NewToken(s.secret, user.ID, user.Email)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	req, err := decode[credentials](r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	user, hash, err := s.store.UserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !auth.CheckPassword(hash, req.Password) {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, err := auth.NewToken(s.secret, user.ID, user.Email)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, currentUser(r))
}
