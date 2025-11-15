package http

import (
	"encoding/json"
	"net/http"
	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/service"
)

type setIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type userResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

type setUserActiveResponse struct {
	User userResponse `json:"user"`
}

type pullRequestShortDTO struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

type userReviewResponse struct {
	UserID       string                `json:"user_id"`
	PullRequests []pullRequestShortDTO `json:"pull_requests"`
}

func newUserResponse(user domain.User) userResponse {
	return userResponse{
		UserID:   string(user.ID),
		Username: user.Username,
		TeamName: string(user.TeamName),
		IsActive: user.IsActive,
	}
}

func newUserReviewResponse(assigns service.UserReviewAssignments) userReviewResponse {
	prs := make([]pullRequestShortDTO, len(assigns.PullRequests))
	for i, pr := range assigns.PullRequests {
		prs[i] = pullRequestShortDTO{
			PullRequestID:   string(pr.ID),
			PullRequestName: pr.Name,
			AuthorID:        string(pr.AuthorID),
			Status:          string(pr.Status),
		}
	}

	return userReviewResponse{
		UserID:       string(assigns.UserID),
		PullRequests: prs,
	}
}

func (h *Handler) handleSetUserActive(w http.ResponseWriter, r *http.Request) {
	var req setIsActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErr := APIError{Code: "BAD_REQUEST", Message: "invalid json body"}
		h.respondJSON(w, r, http.StatusBadRequest, ErrorResponse{Error: apiErr})
		return
	}

	user, err := h.userService.SetIsActive(r.Context(), domain.UserID(req.UserID), req.IsActive)
	if err != nil {
		h.respondError(w, r, err)
		return
	}

	resp := setUserActiveResponse{
		User: newUserResponse(user),
	}

	h.respondJSON(w, r, http.StatusOK, resp)
}

func (h *Handler) handleGetReview(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		apiErr := APIError{Code: "BAD_REQUEST", Message: "missing required 'user_id' query parameter"}
		h.respondJSON(w, r, http.StatusBadRequest, ErrorResponse{Error: apiErr})
		return
	}

	assignments, err := h.userService.ReviewAssignments(r.Context(), domain.UserID(userID))
	if err != nil {
		h.respondError(w, r, err)
		return
	}

	resp := newUserReviewResponse(assignments)

	h.respondJSON(w, r, http.StatusOK, resp)
}
