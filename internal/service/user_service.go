package service

import (
	"context"
	"errors"
	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
	prRepo   repository.PullRequestRepository
}

type UserReviewAssignments struct {
	UserID       domain.UserID
	PullRequests []domain.PullRequestShort
}

func NewUserService(ur repository.UserRepository, prr repository.PullRequestRepository) *UserService {
	return &UserService{
		userRepo: ur,
		prRepo:   prr,
	}
}

func (s *UserService) SetIsActive(ctx context.Context, userID domain.UserID, isActive bool) (domain.User, error) {
	return s.userRepo.SetIsActiveByID(ctx, userID, isActive)
}

func (s *UserService) ReviewAssignments(ctx context.Context, userID domain.UserID) (UserReviewAssignments, error) {
	if _, err := s.userRepo.UserByID(ctx, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return UserReviewAssignments{}, domain.ErrNotFound
		}

		return UserReviewAssignments{}, err
	}

	prs, err := s.prRepo.PullRequestsByReviewer(ctx, userID)
	if err != nil {
		return UserReviewAssignments{}, err
	}

	return UserReviewAssignments{
		UserID:       userID,
		PullRequests: prs,
	}, nil
}
