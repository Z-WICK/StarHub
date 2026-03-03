package api

import (
	"net/http"

	authpkg "github.com/wick/github-star-manager/services/api/internal/auth"
	"github.com/wick/github-star-manager/services/api/internal/config"
	"github.com/wick/github-star-manager/services/api/internal/middleware"
	"github.com/wick/github-star-manager/services/api/internal/response"
	starspkg "github.com/wick/github-star-manager/services/api/internal/stars"
	syncpkg "github.com/wick/github-star-manager/services/api/internal/sync"
)

func NewRouter(
	cfg config.Config,
	authService *authpkg.Service,
	authHandler *authpkg.Handler,
	starsHandler *starspkg.Handler,
	syncHandler *syncpkg.Handler,
) http.Handler {
	publicMux := http.NewServeMux()
	protectedMux := http.NewServeMux()

	publicMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		payload := map[string]string{"status": "ok"}
		response.JSON(w, http.StatusOK, &payload, nil)
	})
	publicMux.HandleFunc("/v1/auth/login", authHandler.Login)
	publicMux.HandleFunc("/v1/auth/session", authHandler.Session)

	protectedMux.HandleFunc("/v1/stars", starsHandler.ListStars)
	protectedMux.HandleFunc("/v1/tags", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			starsHandler.ListTags(w, r)
			return
		}
		if r.Method == http.MethodPost {
			starsHandler.CreateTag(w, r)
			return
		}
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	})
	protectedMux.HandleFunc("/v1/tags/assign", starsHandler.AssignTag)
	protectedMux.HandleFunc("/v1/tags/unassign", starsHandler.UnassignTag)
	protectedMux.HandleFunc("/v1/tags/batch/assign", starsHandler.BatchAssignTag)
	protectedMux.HandleFunc("/v1/tags/batch/unassign", starsHandler.BatchUnassignTag)
	protectedMux.HandleFunc("/v1/notes", starsHandler.UpsertNote)
	protectedMux.HandleFunc("/v1/readme", starsHandler.Readme)
	protectedMux.HandleFunc("/v1/sync", syncHandler.TriggerSync)
	protectedMux.HandleFunc("/v1/sync/status", syncHandler.Status)
	protectedMux.HandleFunc("/v1/sync/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			syncHandler.GetSettings(w, r)
			return
		}
		if r.Method == http.MethodPost {
			syncHandler.UpdateSettings(w, r)
			return
		}
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	})
	protectedMux.HandleFunc("/v1/sync/rules", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			syncHandler.ListRules(w, r)
			return
		}
		if r.Method == http.MethodPost {
			syncHandler.CreateRule(w, r)
			return
		}
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	})
	protectedMux.HandleFunc("/v1/sync/rules/delete", syncHandler.DeleteRule)
	protectedMux.HandleFunc("/v1/sync/rules/apply", syncHandler.ApplyRules)
	protectedMux.HandleFunc("/v1/governance/metrics", syncHandler.GovernanceMetrics)
	protectedMux.HandleFunc("/v1/io/export", syncHandler.ExportData)
	protectedMux.HandleFunc("/v1/io/import", syncHandler.ImportData)

	rootMux := http.NewServeMux()
	rootMux.Handle("/health", publicMux)
	rootMux.Handle("/v1/auth/login", publicMux)
	rootMux.Handle("/v1/auth/session", publicMux)
	rootMux.Handle("/v1/stars", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/tags", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/tags/assign", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/tags/unassign", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/tags/batch/assign", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/tags/batch/unassign", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/notes", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/readme", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/sync", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/sync/status", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/sync/settings", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/sync/rules", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/sync/rules/delete", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/sync/rules/apply", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/governance/metrics", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/io/export", middleware.RequireSession(authService)(protectedMux))
	rootMux.Handle("/v1/io/import", middleware.RequireSession(authService)(protectedMux))

	handler := middleware.CORS(cfg.FrontendOrigin)(rootMux)
	handler = middleware.RateLimit(cfg.RateLimitPerMin)(handler)
	return handler
}
