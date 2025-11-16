package inmemory

import (
	"context"
	"pr-reviewer-service/internal/domain"
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
	if pr, exists := prr.db.PRs[pullRequestID]; exists {
		return pr, nil
	}

	return domain.PullRequest{}, domain.ErrNotFound
}

func (prr *PullRequestRepo) MergeByID(_ context.Context, pullRequestID domain.PullRequestID) (domain.PullRequest, error) {
	if pr, exists := prr.db.PRs[pullRequestID]; exists {
		pr.Status = domain.StatusMerged
		now := time.Now()
		pr.MergedAt = &now
		prr.db.PRs[pullRequestID] = pr

		return pr, nil
	}

	return domain.PullRequest{}, domain.ErrNotFound
}

func (prr *PullRequestRepo) ReassignReviewer(ctx context.Context, pullRequestID domain.PullRequestID, oldUserID domain.UserID, newUserID domain.UserID) (domain.PullRequest, domain.UserID, error) {
	if oldUserID == newUserID {
		return domain.PullRequest{}, domain.UserID(""), domain.ErrNoCandidate
	}

	if pr, exists := prr.db.PRs[pullRequestID]; exists {
		for i, reviewer := range pr.AssignedReviewers {
			if reviewer == oldUserID {
				pr.AssignedReviewers[i] = newUserID
				prr.db.PRs[pullRequestID] = pr
				return pr, newUserID, nil
			}
		}

		return domain.PullRequest{}, domain.UserID(""), domain.ErrNotAssigned
	}

	return domain.PullRequest{}, domain.UserID(""), domain.ErrNotFound
}

func (prr *PullRequestRepo) PullRequestsByReviewer(_ context.Context, userID domain.UserID) ([]domain.PullRequestShort, error) {
	prs := []domain.PullRequestShort{}

	for _, pr := range prr.db.PRs {
		for _, reviewerID := range pr.AssignedReviewers {
			if reviewerID == userID {
				prs = append(prs, domain.PullRequestShort{
					ID:       pr.ID,
					Name:     pr.Name,
					AuthorID: pr.AuthorID,
					Status:   pr.Status,
				})
				break
			}
		}
	}

	return prs, nil
}
