package service

import (
	"context"
	"math/rand"
	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/repository"
	"time"
)

type PullRequestService struct {
	prRepo     repository.PullRequestRepository
	userRepo   repository.UserRepository
	randomizer *rand.Rand
}

func NewPullRequestService(ctx context.Context, prr repository.PullRequestRepository, ur repository.UserRepository) *PullRequestService {
	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &PullRequestService{
		prRepo:     prr,
		userRepo:   ur,
		randomizer: randomizer,
	}
}

func (s *PullRequestService) CreatePR(ctx context.Context, prID domain.PullRequestID, prName string, authorID domain.UserID) (domain.PullRequest, error) {
	author, err := s.userRepo.UserByID(ctx, authorID)
	if err != nil {
		return domain.PullRequest{}, domain.ErrNotFound
	}

	activeMembers, err := s.userRepo.ActiveUsersByTeamName(ctx, author.TeamName)
	if err != nil {
		return domain.PullRequest{}, err
	}

	candidates := make([]domain.UserID, 0, len(activeMembers)-1)
	for _, member := range activeMembers {
		if member.ID != authorID {
			candidates = append(candidates, member.ID)
		}
	}

	reviewers := s.chooseReviewers(candidates)

	pr := domain.PullRequest{
		ID:                prID,
		Name:              prName,
		AuthorID:          authorID,
		Status:            domain.StatusOpen,
		AssignedReviewers: reviewers,
	}

	return s.prRepo.Create(ctx, pr)
}

func (s *PullRequestService) MergePR(ctx context.Context, prID domain.PullRequestID) (domain.PullRequest, error) {
	return s.prRepo.MergeByID(ctx, prID)
}

func (s *PullRequestService) ReassignReviewer(ctx context.Context, prID domain.PullRequestID, oldUserID domain.UserID) (domain.PullRequest, domain.UserID, error) {
	pr, err := s.prRepo.PullRequestByID(ctx, prID)
	if err != nil {
		return domain.PullRequest{}, domain.UserID(""), err
	}

	if pr.Status == domain.StatusMerged {
		return domain.PullRequest{}, domain.UserID(""), domain.ErrPRMerged
	}

	assigned := false
	for _, memberID := range pr.AssignedReviewers {
		if memberID == oldUserID {
			assigned = true
			break
		}
	}
	if !assigned {
		return domain.PullRequest{}, domain.UserID(""), domain.ErrNotAssigned
	}

	oldReviewer, err := s.userRepo.UserByID(ctx, oldUserID)
	if err != nil {
		return domain.PullRequest{}, domain.UserID(""), domain.ErrNotFound
	}

	activeTeamMembers, err := s.userRepo.ActiveUsersByTeamName(ctx, oldReviewer.TeamName)
	if err != nil {
		return domain.PullRequest{}, domain.UserID(""), err
	}

	blacklistedMembers := make(map[domain.UserID]struct{})

	blacklistedMembers[pr.AuthorID] = struct{}{}
	for _, reviewerID := range pr.AssignedReviewers {
		blacklistedMembers[reviewerID] = struct{}{}
	}

	candidates := make([]domain.UserID, 0, len(activeTeamMembers))
	for _, member := range activeTeamMembers {
		if _, exists := blacklistedMembers[member.ID]; !exists {
			candidates = append(candidates, member.ID)
		}
	}

	if len(candidates) == 0 {
		return domain.PullRequest{}, domain.UserID(""), domain.ErrNoCandidate
	}

	newReviewerID := candidates[s.randomizer.Intn(len(candidates))]

	return s.prRepo.ReassignReviewer(ctx, prID, oldUserID, newReviewerID)
}

func (s *PullRequestService) chooseReviewers(candidates []domain.UserID) []domain.UserID {
	candidatesCount := len(candidates)
	if candidatesCount == 0 {
		return []domain.UserID{}
	}

	rand.Shuffle(candidatesCount, func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	return candidates[:min(2, candidatesCount)]
}
