package inmemory

import "pr-reviewer-service/internal/domain"

type InMemoryStorage struct {
	Users             map[domain.UserID]domain.User
	Teams             map[domain.TeamName]domain.Team
	PRs               map[domain.PullRequestID]domain.PullRequest
}

func NewStorage() (*InMemoryStorage, error) {
	return &InMemoryStorage{
		Users:             map[domain.UserID]domain.User{},
		Teams:             map[domain.TeamName]domain.Team{},
		PRs:               map[domain.PullRequestID]domain.PullRequest{},
	}, nil
}
