package http

import (
	"encoding/json"
	"net/http"
	"pr-reviewer-service/internal/domain"
	"time"
)

type createPRRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type pullRequestResponse struct {
	PullRequestID     string   `json:"pull_request_id"`
	PullRequestName   string   `json:"pull_request_name"`
	AuthorID          string   `json:"author_id"`
	Status            string   `json:"status"`
	AssignedReviewers []string `json:"assigned_reviewers"`
	CreatedAt         string   `json:"createdAt"`
	MergedAt          *string  `json:"mergedAt,omitempty"`
}

type createPRResponse struct {
	PR pullRequestResponse `json:"pr"`
}

type mergePRRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

type mergePRResponse struct {
	PR pullRequestResponse `json:"pr"`
}

type reassignPRRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

type reassignPRResponse struct {
	PR         pullRequestResponse `json:"pr"`
	ReplacedBy string              `json:"replaced_by"`
}

func newPullRequestResponse(pr domain.PullRequest) pullRequestResponse {
	reviewers := make([]string, len(pr.AssignedReviewers))
	for i, r := range pr.AssignedReviewers {
		reviewers[i] = string(r)
	}

	var mergedAt *string
	if pr.MergedAt != nil {
		ts := pr.MergedAt.UTC().Format(time.RFC3339)
		mergedAt = &ts
	}

	return pullRequestResponse{
		PullRequestID:     string(pr.ID),
		PullRequestName:   pr.Name,
		AuthorID:          string(pr.AuthorID),
		Status:            string(pr.Status),
		AssignedReviewers: reviewers,
		CreatedAt:         pr.CreatedAt.UTC().Format(time.RFC3339),
		MergedAt:          mergedAt,
	}
}

func (h *Handler) handleCreatePR(w http.ResponseWriter, r *http.Request) {
	var req createPRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErr := APIError{Code: "BAD_REQUEST", Message: "invalid json body"}
		h.respondJSON(w, r, http.StatusBadRequest, ErrorResponse{Error: apiErr})
		return
	}

	pr, err := h.prService.CreatePR(
		r.Context(),
		domain.PullRequestID(req.PullRequestID),
		req.PullRequestName,
		domain.UserID(req.AuthorID),
	)
	if err != nil {
		h.respondError(w, r, err)
		return
	}

	resp := createPRResponse{
		PR: newPullRequestResponse(pr),
	}

	h.respondJSON(w, r, http.StatusCreated, resp)
}

func (h *Handler) handleMergePR(w http.ResponseWriter, r *http.Request) {
	var req mergePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErr := APIError{Code: "BAD_REQUEST", Message: "invalid json body"}
		h.respondJSON(w, r, http.StatusBadRequest, ErrorResponse{Error: apiErr})
		return
	}

	pr, err := h.prService.MergePR(r.Context(), domain.PullRequestID(req.PullRequestID))
	if err != nil {
		h.respondError(w, r, err)
		return
	}

	resp := mergePRResponse{
		PR: newPullRequestResponse(pr),
	}

	h.respondJSON(w, r, http.StatusOK, resp)
}

func (h *Handler) handleReassignPR(w http.ResponseWriter, r *http.Request) {
	var req reassignPRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErr := APIError{Code: "BAD_REQUEST", Message: "invalid json body"}
		h.respondJSON(w, r, http.StatusBadRequest, ErrorResponse{Error: apiErr})
		return
	}

	pr, newUserID, err := h.prService.ReassignReviewer(
		r.Context(),
		domain.PullRequestID(req.PullRequestID),
		domain.UserID(req.OldUserID),
	)
	if err != nil {
		h.respondError(w, r, err)
		return
	}

	resp := reassignPRResponse{
		PR:         newPullRequestResponse(pr),
		ReplacedBy: string(newUserID),
	}

	h.respondJSON(w, r, http.StatusOK, resp)
}
