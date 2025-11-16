package inmemory

import (
	"context"
	"pr-reviewer-service/internal/domain"
	"slices"
	"time"
)

type PullRequestRepo struct {
	db *InMemoryStorage
}

func NewPullRequestRepo(db *InMemoryStorage) *PullRequestRepo {
	return &PullRequestRepo{
		db: db,
	}
}

func (prr *PullRequestRepo) Create(_ context.Context, pr domain.PullRequest) (domain.PullRequest, error) {
	if _, exists := prr.db.PRs[pr.ID]; exists {
		return domain.PullRequest{}, domain.ErrPRExists
	}

	pr.CreatedAt = time.Now()
	prr.db.PRs[pr.ID] = pr

	return pr, nil
}

func (prr *PullRequestRepo) PullRequestByID(_ context.Context, pullRequestID domain.PullRequestID) (domain.PullRequest, error) {
	pr, exists := prr.db.PRs[pullRequestID]
	if !exists {
		return domain.PullRequest{}, domain.ErrNotFound
	}

	return pr, nil
}

func (prr *PullRequestRepo) MergeByID(_ context.Context, pullRequestID domain.PullRequestID) (domain.PullRequest, error) {
	pr, exists := prr.db.PRs[pullRequestID]
	if !exists {
		return domain.PullRequest{}, domain.ErrNotFound
	}

	pr.Status = domain.StatusMerged

	if pr.MergedAt != nil {
		now := time.Now()
		pr.MergedAt = &now
	}

	prr.db.PRs[pullRequestID] = pr

	return pr, nil
}

func (prr *PullRequestRepo) ReassignReviewer(ctx context.Context, pullRequestID domain.PullRequestID, oldUserID domain.UserID, newUserID domain.UserID) (domain.PullRequest, domain.UserID, error) {
	if oldUserID == newUserID {
		return domain.PullRequest{}, domain.UserID(""), domain.ErrNoCandidate
	}

	pr, exists := prr.db.PRs[pullRequestID]
	if !exists {
		return domain.PullRequest{}, domain.UserID(""), domain.ErrNotFound
	}

	for i, reviewer := range pr.AssignedReviewers {
		if reviewer == oldUserID {
			pr.AssignedReviewers[i] = newUserID
			prr.db.PRs[pullRequestID] = pr
			return pr, newUserID, nil
		}
	}

	return domain.PullRequest{}, domain.UserID(""), domain.ErrNotAssigned
}

func (prr *PullRequestRepo) PullRequestsByReviewer(_ context.Context, userID domain.UserID) ([]domain.PullRequestShort, error) {
	prs := []domain.PullRequestShort{}

	for _, pr := range prr.db.PRs {
		if slices.Contains(pr.AssignedReviewers, userID) {
			prs = append(prs, domain.PullRequestShort{
				ID:       pr.ID,
				Name:     pr.Name,
				AuthorID: pr.AuthorID,
				Status:   pr.Status,
			})
		}
	}

	return prs, nil
}
