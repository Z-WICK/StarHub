package sync

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/wick/github-star-manager/services/api/internal/db"
	"github.com/wick/github-star-manager/services/api/internal/middleware"
	"github.com/wick/github-star-manager/services/api/internal/response"
)

type Handler struct {
	service *Service
}

type updateSyncSettingsRequest struct {
	Enabled       bool `json:"enabled"`
	IntervalHours int  `json:"intervalHours"`
	RetryMax      int  `json:"retryMax"`
}

type createSmartRuleRequest struct {
	Name                string `json:"name"`
	Enabled             bool   `json:"enabled"`
	LanguageEquals      string `json:"languageEquals"`
	OwnerContains       string `json:"ownerContains"`
	NameContains        string `json:"nameContains"`
	DescriptionContains string `json:"descriptionContains"`
	TagID               int64  `json:"tagId"`
}

type deleteSmartRuleRequest struct {
	RuleID int64 `json:"ruleId"`
}

type applyRulesResponse struct {
	Applied int64 `json:"applied"`
}

type importPayloadRequest struct {
	SyncSettings *db.SyncSettings      `json:"syncSettings"`
	Tags         []db.Tag              `json:"tags"`
	SmartRules   []db.ExportRule       `json:"smartRules"`
	Notes        []db.ExportNote       `json:"notes"`
	TagBindings  []db.ExportTagBinding `json:"tagBindings"`
}

const maxJSONBodyBytes int64 = 64 * 1024

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, out any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("invalid request body")
	}
	return nil
}

func (h *Handler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	result, err := h.service.SyncStars(r.Context(), userID)
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			response.Error(w, http.StatusConflict, "sync already running")
			return
		}
		response.Error(w, http.StatusInternalServerError, "sync failed")
		return
	}
	response.JSON(w, http.StatusOK, &result, nil)
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	result, err := h.service.LatestSyncStatus(r.Context(), userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			empty := db.SyncJob{}
			response.JSON(w, http.StatusOK, &empty, nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to load sync status")
		return
	}
	response.JSON(w, http.StatusOK, &result, nil)
}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	result, err := h.service.GetSyncSettings(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to load settings")
		return
	}
	response.JSON(w, http.StatusOK, &result, nil)
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	var body updateSyncSettingsRequest
	if err := decodeJSONBody(w, r, &body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.UpdateSyncSettings(r.Context(), userID, body.Enabled, body.IntervalHours, body.RetryMax)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, &result, nil)
}

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	items, err := h.service.ListSmartRules(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to load rules")
		return
	}
	response.JSON(w, http.StatusOK, &items, nil)
}

func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	var body createSmartRuleRequest
	if err := decodeJSONBody(w, r, &body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created, err := h.service.CreateSmartRule(r.Context(), userID, db.SmartRule{
		Name:                body.Name,
		Enabled:             body.Enabled,
		LanguageEquals:      body.LanguageEquals,
		OwnerContains:       body.OwnerContains,
		NameContains:        body.NameContains,
		DescriptionContains: body.DescriptionContains,
		TagID:               body.TagID,
	})
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			response.Error(w, http.StatusBadRequest, "tag not found")
			return
		}
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, &created, nil)
}

func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	var body deleteSmartRuleRequest
	if err := decodeJSONBody(w, r, &body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.DeleteSmartRule(r.Context(), userID, body.RuleID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "rule not found")
			return
		}
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	ack := map[string]string{"status": "ok"}
	response.JSON(w, http.StatusOK, &ack, nil)
}

func (h *Handler) GovernanceMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	result, err := h.service.GetGovernanceMetrics(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to load governance metrics")
		return
	}
	response.JSON(w, http.StatusOK, &result, nil)
}

func (h *Handler) ExportData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	result, err := h.service.ExportData(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "export failed")
		return
	}
	response.JSON(w, http.StatusOK, &result, nil)
}

func (h *Handler) ImportData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	var body importPayloadRequest
	if err := decodeJSONBody(w, r, &body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.service.ImportData(r.Context(), userID, db.ImportPayload{
		SyncSettings: body.SyncSettings,
		Tags:         body.Tags,
		SmartRules:   body.SmartRules,
		Notes:        body.Notes,
		TagBindings:  body.TagBindings,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, "import failed")
		return
	}
	response.JSON(w, http.StatusOK, &result, nil)
}

func (h *Handler) ApplyRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid user")
		return
	}

	affected, err := h.service.ApplyRulesNow(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "apply rules failed")
		return
	}
	payload := applyRulesResponse{Applied: affected}
	response.JSON(w, http.StatusOK, &payload, nil)
}
