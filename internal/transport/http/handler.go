package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/service"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

type Handler struct {
	teamService *service.TeamService
	userService *service.UserService
	prService   *service.PullRequestService
	logger      *slog.Logger
}

func NewHandeler(ts *service.TeamService, us *service.UserService, prs *service.PullRequestService, logger *slog.Logger) *Handler {
	return &Handler{
		teamService: ts,
		userService: us,
		prService:   prs,
		logger:      logger,
	}
}

func (h *Handler) respondJSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to write json responce", "error", err)
	}
}

func (h *Handler) respondError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	apiErr := APIError{
		Code:    "INTERNAL_ERROR",
		Message: "unknown error",
	}

	if errors.Is(err, domain.ErrNotFound) {
		status = http.StatusNotFound
		apiErr = APIError{Code: "NOT_FOUND", Message: err.Error()}
	} else if errors.Is(err, domain.ErrTeamExists) {
		status = http.StatusBadRequest
		apiErr = APIError{Code: "TEAM_EXISTS", Message: err.Error()}
	} else if errors.Is(err, domain.ErrPRExists) {
		status = http.StatusConflict
		apiErr = APIError{Code: "PR_EXISTS", Message: err.Error()}
	} else if errors.Is(err, domain.ErrTeamExists) {
		status = http.StatusConflict
		apiErr = APIError{Code: "PR_MERGED", Message: err.Error()}
	} else if errors.Is(err, domain.ErrTeamExists) {
		status = http.StatusConflict
		apiErr = APIError{Code: "NOT_ASSIGNED", Message: err.Error()}
	} else if errors.Is(err, domain.ErrTeamExists) {
		status = http.StatusConflict
		apiErr = APIError{Code: "NO_CANDIDATE", Message: err.Error()}
	}

	if status == http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "http server error", "error", err)
	}

	h.respondJSON(w, r, status, ErrorResponse{Error: apiErr})
}

func NewSlogLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t1 := time.Now()

			next.ServeHTTP(ww, r)

			logger.InfoContext(r.Context(), "request served",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(t1).Milliseconds(),
				"bytes_written", ww.BytesWritten(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		}

		return http.HandlerFunc(fn)
	}
}
