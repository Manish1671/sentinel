package httpapi

import (
	"net/http"
)

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, s.log, err)
		return
	}
	result, err := s.auth.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	s.setSessionCookie(w, result.Token, result.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"token": result.Token,
			"user": map[string]any{
				"id":           result.User.ID,
				"email":        result.User.Email,
				"display_name": result.User.DisplayName,
				"role":         result.User.Role,
			},
		},
	})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.auth.Logout(r.Context(), sessionToken(r)); err != nil {
		writeError(w, r, s.log, err)
		return
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":           user.ID,
			"email":        user.Email,
			"display_name": user.DisplayName,
			"role":         user.Role,
			"status":       user.Status,
		},
	})
}
