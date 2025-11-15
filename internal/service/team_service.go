package service

import (
	"context"
	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/repository"
)

type TeamService struct {
	teamRepo repository.TeamRepository
	userRepo repository.UserRepository
}

func NewTeamService(tr repository.TeamRepository, ur repository.UserRepository) *TeamService {
	return &TeamService{
		teamRepo: tr,
		userRepo: ur,
	}
}

func (s *TeamService) CreateTeam(ctx context.Context, team domain.Team) error {
	return s.teamRepo.Create(ctx, team)
}

func (s *TeamService) Team(ctx context.Context, teamName domain.TeamName) (domain.Team, error) {
	return s.teamRepo.TeamByName(ctx, teamName)
}
