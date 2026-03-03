package stars

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/wick/github-star-manager/services/api/internal/db"
	"github.com/wick/github-star-manager/services/api/internal/github"
	"github.com/wick/github-star-manager/services/api/internal/middleware"
	"github.com/wick/github-star-manager/services/api/internal/response"
)

type Handler struct {
	service *Service
}

type listPayload struct {
	Items []db.StarRecord `json:"items"`
}

type noteRequest struct {
	RepositoryID int64  `json:"repositoryId"`
	Content      string `json:"content"`
}

type createTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type readmeRequest struct {
	RepositoryID int64 `json:"repositoryId"`
}

type assignTagRequest struct {
	RepositoryID int64 `json:"repositoryId"`
	TagID        int64 `json:"tagId"`
}

type batchTagRequest struct {
	RepositoryIDs []int64 `json:"repositoryIds"`
	TagID         int64   `json:"tagId"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListStars(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	tagID, _ := strconv.ParseInt(r.URL.Query().Get("tagId"), 10, 64)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		q = strings.TrimSpace(r.URL.Query().Get("query"))
	}

	var hasNote *bool
	hasNoteRaw := strings.TrimSpace(r.URL.Query().Get("hasNote"))
	if hasNoteRaw != "" {
		parsed, err := strconv.ParseBool(hasNoteRaw)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid hasNote")
			return
		}
		hasNote = &parsed
	}

	filters := db.StarFilters{
		Query:     q,
		Language:  strings.TrimSpace(r.URL.Query().Get("language")),
		TagID:     tagID,
		HasNote:   hasNote,
		SortBy:    strings.TrimSpace(r.URL.Query().Get("sortBy")),
		SortOrder: strings.TrimSpace(r.URL.Query().Get("sortOrder")),
		Page:      page,
		Limit:     limit,
	}

	result, err := h.service.List(r.Context(), userID, filters)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list stars")
		return
	}

	payload := listPayload{Items: result.Items}
	meta := response.PageMeta{Page: filters.Page, Limit: filters.Limit, Total: result.Total}
	if meta.Page <= 0 {
		meta.Page = 1
	}
	if meta.Limit <= 0 || meta.Limit > 100 {
		meta.Limit = 20
	}
	response.JSON(w, http.StatusOK, &payload, &meta)
}

func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}
	var body createTagRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tag, err := h.service.CreateTag(r.Context(), userID, body.Name, body.Color)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, &tag, nil)
}

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}
	items, err := h.service.ListTags(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list tags")
		return
	}
	response.JSON(w, http.StatusOK, &items, nil)
}

func (h *Handler) AssignTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}
	var body assignTagRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.service.AssignTag(r.Context(), userID, body.RepositoryID, body.TagID); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ack := map[string]string{"status": "ok"}
	response.JSON(w, http.StatusOK, &ack, nil)
}

func (h *Handler) UnassignTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}
	var body assignTagRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.service.UnassignTag(r.Context(), userID, body.RepositoryID, body.TagID); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ack := map[string]string{"status": "ok"}
	response.JSON(w, http.StatusOK, &ack, nil)
}

func (h *Handler) BatchAssignTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}
	var body batchTagRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.service.BatchAssignTag(r.Context(), userID, body.RepositoryIDs, body.TagID); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ack := map[string]string{"status": "ok"}
	response.JSON(w, http.StatusOK, &ack, nil)
}

func (h *Handler) BatchUnassignTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}
	var body batchTagRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.service.BatchUnassignTag(r.Context(), userID, body.RepositoryIDs, body.TagID); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ack := map[string]string{"status": "ok"}
	response.JSON(w, http.StatusOK, &ack, nil)
}

func (h *Handler) UpsertNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}
	var body noteRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.service.UpsertNote(r.Context(), userID, body.RepositoryID, body.Content); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ack := map[string]string{"status": "ok"}
	response.JSON(w, http.StatusOK, &ack, nil)
}

func (h *Handler) Readme(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	var body readmeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.GetReadme(r.Context(), userID, body.RepositoryID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) || errors.Is(err, github.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "readme not found")
			return
		}
		response.Error(w, http.StatusBadGateway, "failed to load readme")
		return
	}

	response.JSON(w, http.StatusOK, &result, nil)
}
