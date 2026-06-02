package webapi

import (
	"encoding/json"
	"net/http"
)

type response struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response{OK: true, Data: data})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response{OK: false, Error: msg})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

type authEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

type createRouteFileRequest struct {
	Name string `json:"name"`
}

type setActiveRouteRequest struct {
	Name string `json:"name"`
}

type logsQuery struct {
	N int `json:"n"`
}

type checkResponse struct {
	Valid bool `json:"valid"`
}
