package postgres

import (
	"context"
	"errors"
	"pr-reviewer-service/internal/domain"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PullRequestRepo struct {
	db *pgxpool.Pool
}

func NewPullRequestRepo(db *pgxpool.Pool) *PullRequestRepo {
	return &PullRequestRepo{
		db: db,
	}
}

func (prr *PullRequestRepo) Create(ctx context.Context, pr domain.PullRequest) (domain.PullRequest, error) {
	tx, err := prr.db.Begin(ctx)
	if err != nil {
		return domain.PullRequest{}, err
	}
	defer tx.Rollback(ctx)

	createPRQuery := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`

	var createdAt time.Time
	err = tx.QueryRow(ctx, createPRQuery, pr.ID, pr.Name, pr.AuthorID, pr.Status).
		Scan(&createdAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.PullRequest{}, domain.ErrPRExists
		}

		return domain.PullRequest{}, err
	}

	pr.CreatedAt = createdAt

	if len(pr.AssignedReviewers) > 0 {
		insertReviewersQuery := `
			INSERT INTO pull_request_reviewers (pull_request_id, user_id)
			VALUES ($1, $2)
		`

		batch := &pgx.Batch{}
		for _, reviewerID := range pr.AssignedReviewers {
			batch.Queue(insertReviewersQuery, pr.ID, reviewerID)
		}

		batchRes := tx.SendBatch(ctx, batch)
		if err := batchRes.Close(); err != nil {
			return domain.PullRequest{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.PullRequest{}, err
	}

	return pr, nil
}

func (prr *PullRequestRepo) PullRequestByID(ctx context.Context, pullRequestID domain.PullRequestID) (domain.PullRequest, error) {
	return prr.pullRequestByID(ctx, prr.db, pullRequestID)
}

func (prr *PullRequestRepo) MergeByID(ctx context.Context, pullRequestID domain.PullRequestID) (domain.PullRequest, error) {
	tx, err := prr.db.Begin(ctx)
	if err != nil {
		return domain.PullRequest{}, err
	}
	defer tx.Rollback(ctx)

	mergeQuery := `
		UPDATE pull_requests
		SET
			status = 'MERGED',
			merged_at = COALESCE(merged_at, NOW())
		WHERE pull_request_id = $1
	`

	_, err = tx.Exec(ctx, mergeQuery, pullRequestID)
	if err != nil {
		return domain.PullRequest{}, err
	}

	pullRequest, err := prr.pullRequestByID(ctx, tx, pullRequestID)
	if err != nil {
		return domain.PullRequest{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.PullRequest{}, err
	}

	return pullRequest, nil
}

func (prr *PullRequestRepo) ReassignReviewer(ctx context.Context, pullRequestID domain.PullRequestID, oldUserID domain.UserID, newUserID domain.UserID) (domain.PullRequest, domain.UserID, error) {
	if oldUserID == newUserID {
		return domain.PullRequest{}, domain.UserID(""), domain.ErrNoCandidate
	}

	tx, err := prr.db.Begin(ctx)
	if err != nil {
		return domain.PullRequest{}, domain.UserID(""), err
	}
	defer tx.Rollback(ctx)

	sqlReassign := `
		UPDATE pull_request_reviewers
		SET user_id = $1
		WHERE pull_request_id = $2 AND user_id = $3
	`
	tag, err := tx.Exec(ctx, sqlReassign, newUserID, pullRequestID, oldUserID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.PullRequest{}, domain.UserID(""), domain.ErrPRExists
		}
		return domain.PullRequest{}, domain.UserID(""), err
	}

	if tag.RowsAffected() == 0 {
		var prExists bool

		err := tx.QueryRow(ctx, "SELECT TRUE FROM pull_requests WHERE pull_request_id = $1", pullRequestID).Scan(&prExists)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.PullRequest{}, domain.UserID(""), domain.ErrNotFound
			}
			return domain.PullRequest{}, domain.UserID(""), err
		}

		return domain.PullRequest{}, domain.UserID(""), domain.ErrNotAssigned
	}

	pr, err := prr.pullRequestByID(ctx, tx, pullRequestID)
	if err != nil {
		return domain.PullRequest{}, domain.UserID(""), err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.PullRequest{}, domain.UserID(""), err
	}

	return pr, newUserID, nil
}

func (prr *PullRequestRepo) PullRequestsByReviewer(ctx context.Context, userID domain.UserID) ([]domain.PullRequestShort, error) {
	prByReviewerQuery := `
		SELECT 
			pr.pull_request_id,
			pr.pull_request_name,
			pr.author_id,
			pr.status
		FROM pull_requests pr
		JOIN pull_request_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE prr.user_id = $1
		ORDER BY pr.created_at DESC
	`

	rows, err := prr.db.Query(ctx, prByReviewerQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []domain.PullRequestShort
	for rows.Next() {
		var pr domain.PullRequestShort
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return prs, nil
}

type RowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (prr *PullRequestRepo) pullRequestByID(ctx context.Context, rq RowQuerier, pullRequestID domain.PullRequestID) (domain.PullRequest, error) {
	prByIDQuery := `
		SELECT
			pr.pull_request_id,
			pr.pull_request_name,
			pr.author_id,
			pr.status,
			pr.created_at,
			pr.merged_at,
			COALESCE(ARRAY_AGG(prr.user_id) FILTER (WHERE prr.user_id IS NOT NULL), '{}') AS assigned_reviewers
		FROM pull_requests pr
		LEFT JOIN pull_request_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE pr.pull_request_id = $1
		GROUP BY pr.pull_request_id
	`

	var pr domain.PullRequest
	var reviewers []domain.UserID

	err := rq.QueryRow(ctx, prByIDQuery, pullRequestID).Scan(
		&pr.ID,
		&pr.Name,
		&pr.AuthorID,
		&pr.Status,
		&pr.CreatedAt,
		&pr.MergedAt,
		&reviewers,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PullRequest{}, domain.ErrNotFound
		}
		return domain.PullRequest{}, err
	}

	pr.AssignedReviewers = reviewers

	return pr, nil
}
