package inmemory_test

import (
	"context"
	"fmt"
	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/repository/inmemory"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEnviroment struct {
	ctx     context.Context
	storage *inmemory.InMemoryStorage
	prRepo  *inmemory.PullRequestRepo
}

func setup() testEnviroment {
	storage, _ := inmemory.NewStorage()
	prRepo := inmemory.NewPullRequestRepo(storage)

	return testEnviroment{
		ctx:     context.Background(),
		storage: storage,
		prRepo:  prRepo,
	}
}

var (
	authorID         = domain.UserID("u-author")
	firstReviewerID  = domain.UserID("u-reviewer-first")
	secondReviewerID = domain.UserID("u-reviwer-second")
	prID             = domain.PullRequestID("pr-1")

	testPR = domain.PullRequest{
		ID:                prID,
		Name:              "Test PR",
		AuthorID:          authorID,
		Status:            domain.StatusOpen,
		AssignedReviewers: []domain.UserID{firstReviewerID},
	}
)

func TestSuccessCreatePR(t *testing.T) {
	e := setup()

	pr, err := e.prRepo.Create(e.ctx, testPR)
	require.NoError(t, err)
	assert.Equal(t, prID, pr.ID)
	assert.NotNil(t, pr.CreatedAt)

	dbPR, exists := e.storage.PRs[prID]
	require.True(t, exists)
	assert.Equal(t, "Test PR", dbPR.Name)
	assert.Equal(t, firstReviewerID, dbPR.AssignedReviewers[0])
}

func TestFailCreatePRWhenAlreadyExist(t *testing.T) {
	e := setup()
	_, err := e.prRepo.Create(e.ctx, testPR)
	require.NoError(t, err)

	prDuplicate := testPR
	prDuplicate.Name = "another-pr"

	_, err = e.prRepo.Create(e.ctx, prDuplicate)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrPRExists)
}

func TestSuccessPullRequestByID(t *testing.T) {
	h := setup()
	h.storage.PRs[prID] = testPR

	pr, err := h.prRepo.PullRequestByID(h.ctx, prID)
	require.NoError(t, err)
	assert.Equal(t, prID, pr.ID)
	assert.Equal(t, "Test PR", pr.Name)
}

func TestFailPullRequestByIDWhenNotFound(t *testing.T) {
	e := setup()

	_, err := e.prRepo.PullRequestByID(e.ctx, "non-existent-pr")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSuccessMergeByID(t *testing.T) {
	e := setup()
	e.storage.PRs[prID] = testPR

	mergedPR, err := e.prRepo.MergeByID(e.ctx, prID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusMerged, mergedPR.Status)
	assert.NotNil(t, mergedPR.MergedAt)

	dbPR := e.storage.PRs[prID]
	assert.Equal(t, domain.StatusMerged, dbPR.Status)
	assert.NotNil(t, dbPR.MergedAt)
}

func TestMergeByIDIsIdempotent(t *testing.T) {
	e := setup()
	e.storage.PRs[prID] = testPR

	mergedPR1, err := e.prRepo.MergeByID(e.ctx, prID)
	require.NoError(t, err)

	firstMergeTime := mergedPR1.MergedAt
	require.NotNil(t, firstMergeTime)

	time.Sleep(1 * time.Millisecond) // write time provider

	mergedPR2, err := e.prRepo.MergeByID(e.ctx, prID)
	require.NoError(t, err)

	assert.Equal(t, firstMergeTime, mergedPR2.MergedAt)
	assert.Equal(t, firstMergeTime.UnixNano(), mergedPR2.MergedAt.UnixNano())
}

func TestFailMergeByIDWhenNotFound(t *testing.T) {
	e := setup()

	_, err := e.prRepo.MergeByID(e.ctx, "non-existent-pr")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSuccessReassignReviewer(t *testing.T) {
	e := setup()
	e.storage.PRs[prID] = testPR

	pr, newReviewer, err := e.prRepo.ReassignReviewer(e.ctx, prID, firstReviewerID, secondReviewerID)
	require.NoError(t, err)
	assert.Equal(t, secondReviewerID, newReviewer)
	assert.Len(t, pr.AssignedReviewers, 1)
	assert.Equal(t, secondReviewerID, pr.AssignedReviewers[0])

	dbPR := e.storage.PRs[prID]
	assert.Equal(t, secondReviewerID, dbPR.AssignedReviewers[0])
}

func TestFailReassignReviewerOnSameUser(t *testing.T) {
	e := setup()
	e.storage.PRs[prID] = testPR

	_, _, err := e.prRepo.ReassignReviewer(e.ctx, prID, firstReviewerID, firstReviewerID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNoCandidate)
}

func TestFailReassignReviewerWhenOldIsNotAssigned(t *testing.T) {
	e := setup()
	e.storage.PRs[prID] = testPR

	_, _, err := e.prRepo.ReassignReviewer(e.ctx, prID, secondReviewerID, authorID)
	fmt.Print(err)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotAssigned)
}

func TestSucessPullRequestsByReviewer(t *testing.T) {
	e := setup()
	pr2 := domain.PullRequest{ID: "pr-2", AssignedReviewers: []domain.UserID{firstReviewerID}}
	pr3 := domain.PullRequest{ID: "pr-3", AssignedReviewers: []domain.UserID{secondReviewerID}}
	pr4 := domain.PullRequest{ID: "pr-4", AssignedReviewers: []domain.UserID{firstReviewerID, secondReviewerID}}

	e.storage.PRs[pr2.ID] = pr2
	e.storage.PRs[pr3.ID] = pr3
	e.storage.PRs[pr4.ID] = pr4

	prs, err := e.prRepo.PullRequestsByReviewer(e.ctx, firstReviewerID)
	require.NoError(t, err)
	require.Len(t, prs, 2)

	prIDs := []domain.PullRequestID{prs[0].ID, prs[1].ID}
	assert.Contains(t, prIDs, domain.PullRequestID("pr-2"))
	assert.Contains(t, prIDs, domain.PullRequestID("pr-4"))
	assert.NotContains(t, prIDs, domain.PullRequestID("pr-3"))
}
