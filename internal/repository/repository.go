package repository

import (
	"context"
	"pr-reviewer-service/internal/domain"
)

type TeamRepository interface {
	Create(ctx context.Context, team domain.Team) error
	TeamByName(ctx context.Context, teamName domain.TeamName) (domain.Team, error)
}

type UserRepository interface {
	Create(ctx context.Context, user domain.User) error
	UserByID(ctx context.Context, userID domain.UserID) (domain.User, error)
	SetIsActiveByID(ctx context.Context, userID domain.UserID, isActive bool) (domain.User, error)
	ActiveUsersByTeamName(ctx context.Context, teamName domain.TeamName) ([]domain.User, error)
}

type PullRequestRepository interface {
	Create(ctx context.Context, pullRequest domain.PullRequest) (domain.PullRequest, error)
	PullRequestByID(ctx context.Context, pullRequestID domain.PullRequestID) (domain.PullRequest, error)
	MergeByID(ctx context.Context, pullRequestID domain.PullRequestID) (domain.PullRequest, error)
	ReassignReviewer(ctx context.Context, pullRequestID domain.PullRequestID, oldUserID domain.UserID, newUserID domain.UserID) (domain.PullRequest, error)
	PullRequestsByReviewer(ctx context.Context, userID domain.UserID) ([]domain.PullRequestShort, error)
}
