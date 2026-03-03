package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/wick/github-star-manager/services/api/internal/response"
)

type Handler struct {
	service *Service
}

type loginRequest struct {
	GitHubToken string `json:"githubToken"`
}

type sessionResponse struct {
	UserID int64 `json:"userId"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body loginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	body.GitHubToken = strings.TrimSpace(body.GitHubToken)
	if body.GitHubToken == "" {
		response.Error(w, http.StatusBadRequest, "github token is required")
		return
	}

	result, err := h.service.LoginWithGitHubToken(r.Context(), body.GitHubToken)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "github authentication failed")
		return
	}

	response.JSON(w, http.StatusOK, &result, nil)
}

func (h *Handler) Session(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		response.Error(w, http.StatusUnauthorized, "missing authorization")
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

	userID, err := h.service.ValidateSessionToken(r.Context(), token)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid session")
		return
	}

	payload := sessionResponse{UserID: userID}
	response.JSON(w, http.StatusOK, &payload, nil)
}
