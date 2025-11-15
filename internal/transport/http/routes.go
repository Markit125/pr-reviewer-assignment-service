package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (h *Handler) RegisterRoutes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(NewSlogLogger(h.logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	r.Route("/team", func(r chi.Router) {
		r.Post("/add", h.handlerAddTeam)
		r.Get("/get", h.handleGetTeam)
	})

	r.Route("/users", func(r chi.Router) {
		r.Post("/setIsActive", h.handleSetUserActive)
		r.Get("/getReview", h.handleGetReview)
	})

	r.Route("/pullRequest", func(r chi.Router) {
		r.Post("/create", h.handleCreatePR)
		r.Post("/merge", h.handleMergePR)
		r.Post("/reassign", h.handleReassignPR)
	})

	r.Get("/health", h.handleHealthCheck)

	return r
}

func (h *Handler) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}
