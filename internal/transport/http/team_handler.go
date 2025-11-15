package http

import (
	"encoding/json"
	"net/http"
	"pr-reviewer-service/internal/domain"
)

type teamMemberDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type teamRequest struct {
	TeamName string          `json:"team_name"`
	Members  []teamMemberDTO `json:"members"`
}

type teamResponse struct {
	TeamName string          `json:"team_name"`
	Members  []teamMemberDTO `json:"members"`
}

type teamAddResponse struct {
	Team teamResponse `json:"team"`
}

func (req *teamRequest) toDomainTeam() domain.Team {
	members := make([]domain.TeamMember, len(req.Members))
	for i, m := range req.Members {
		members[i] = domain.TeamMember{
			UserID:   domain.UserID(m.UserID),
			Username: m.Username,
			IsActive: m.IsActive,
		}
	}
	return domain.Team{
		Name:    domain.TeamName(req.TeamName),
		Members: members,
	}
}

func newTeamResponse(team domain.Team) teamResponse {
	members := make([]teamMemberDTO, len(team.Members))
	for i, m := range team.Members {
		members[i] = teamMemberDTO{
			UserID:   string(m.UserID),
			Username: m.Username,
			IsActive: m.IsActive,
		}
	}
	return teamResponse{
		TeamName: string(team.Name),
		Members:  members,
	}
}

func (h *Handler) handlerAddTeam(w http.ResponseWriter, r *http.Request) {
	var req teamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErr := APIError{Code: "BAD_REQUEST", Message: "invalid json body"}
		h.respondJSON(w, r, http.StatusBadRequest, ErrorResponse{Error: apiErr})
		return
	}

	team := req.toDomainTeam()

	if err := h.teamService.CreateTeam(r.Context(), team); err != nil {
		h.respondError(w, r, err)
	}

	resp := teamAddResponse{
		Team: newTeamResponse(team),
	}

	h.respondJSON(w, r, http.StatusCreated, resp)
}

func (h *Handler) handleGetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		apiErr := APIError{Code: "BAD_REQUEST", Message: "missing required 'team_name' query parameter"}
		h.respondJSON(w, r, http.StatusBadRequest, ErrorResponse{Error: apiErr})
		return
	}

	team, err := h.teamService.Team(r.Context(), domain.TeamName(teamName))
	if err != nil {
		h.respondError(w, r, err)
		return
	}

	resp := newTeamResponse(team)

	h.respondJSON(w, r, http.StatusOK, resp)
}
