package postgres

import (
	"context"
	"errors"
	"fmt"
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

		batchRes := prr.db.SendBatch(ctx, batch)
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
			status = 'MERGED'
			merged_at = COALESCE(merged_at, NOW())
		WHERE pull_request_id = $1
	`

	tag, err := tx.Exec(ctx, mergeQuery, pullRequestID)
	if err != nil {
		return domain.PullRequest{}, err
	}

	if tag.RowsAffected() == 0 {
		exists := false

		isPullRequestExistQuery := `SELECT TRUE FROM pull_requests WHERE pull_request_id = $1`
		err := tx.QueryRow(ctx, isPullRequestExistQuery, pullRequestID).Scan(&exists)
		if err != nil && errors.Is(err, pgx.ErrNoRows) {
			return domain.PullRequest{}, err
		}

		if !exists {
			return domain.PullRequest{}, domain.ErrNotFound
		}
	}

	pullRequest, err := prr.pullRequestByID(ctx, tx, pullRequestID)
	if err != nil {
		return domain.PullRequest{}, err
	}

	return pullRequest, err
}

func (prr *PullRequestRepo) ReassignReviewer(ctx context.Context, pullRequestID domain.PullRequestID, oldUserID domain.UserID, newUserID domain.UserID) (domain.PullRequest, error) {
	if oldUserID == newUserID {
		return domain.PullRequest{}, fmt.Errorf("trying to change reviewer on the same reviewer with id: %s", newUserID)
	}

	tx, err := prr.db.Begin(ctx)
	if err != nil {
		return domain.PullRequest{}, err
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
			return domain.PullRequest{}, domain.ErrPRExists
		}
		return domain.PullRequest{}, err
	}

	if tag.RowsAffected() == 0 {
		var prExists, userAssigned bool

		prExistsErr := tx.QueryRow(ctx, "SELECT TRUE FROM pull_requests WHERE pull_request_id = $1", pullRequestID).Scan(&prExists)
		if prExistsErr != nil && !errors.Is(prExistsErr, pgx.ErrNoRows) {
			return domain.PullRequest{}, prExistsErr
		}
		if !prExists {
			return domain.PullRequest{}, domain.ErrNotFound
		}

		userAssignedErr := tx.QueryRow(ctx, "SELECT TRUE FROM pull_request_reviewers WHERE pull_request_id = $1 AND user_id = $2", pullRequestID, oldUserID).Scan(&userAssigned)
		if userAssignedErr != nil && !errors.Is(userAssignedErr, pgx.ErrNoRows) {
			return domain.PullRequest{}, userAssignedErr
		}
		if !userAssigned {
			return domain.PullRequest{}, domain.ErrNotAssigned
		}

		return domain.PullRequest{}, errors.New("reassignment failed for an unknown reason")
	}

	pr, err := prr.pullRequestByID(ctx, tx, pullRequestID)
	if err != nil {
		return domain.PullRequest{}, err
	}

	return pr, tx.Commit(ctx)
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
