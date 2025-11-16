package service_test

import (
	"context"
	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/repository/inmemory"
	"pr-reviewer-service/internal/service"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPREnviroment struct {
	ctx     context.Context
	storage *inmemory.InMemoryStorage

	userRepo repository.UserRepository
	teamRepo repository.TeamRepository
	prRepo   repository.PullRequestRepository

	prService   *service.PullRequestService
	teamService *service.TeamService
}

func setup() testPREnviroment {
	storage, _ := inmemory.NewStorage()

	userRepo := inmemory.NewUserRepo(storage)
	teamRepo := inmemory.NewTeamRepo(storage)
	prRepo := inmemory.NewPullRequestRepo(storage)

	teamService := service.NewTeamService(teamRepo, userRepo)
	prService := service.NewPullRequestService(prRepo, userRepo)

	return testPREnviroment{
		ctx:         context.Background(),
		storage:     storage,
		userRepo:    userRepo,
		teamRepo:    teamRepo,
		prRepo:      prRepo,
		prService:   prService,
		teamService: teamService,
	}
}

var (
	teamName         = domain.TeamName("backend")
	authorID         = domain.UserID("u-author")
	firstReviewerID  = domain.UserID("u-reviewer-1")
	secondReviewerID = domain.UserID("u-reviewer-2")
	inactiveUserID   = domain.UserID("u-inactive")

	testTeam = domain.Team{
		Name: teamName,
		Members: []domain.TeamMember{
			{UserID: authorID, Username: "Author", IsActive: true},
			{UserID: firstReviewerID, Username: "Reviewer 1", IsActive: true},
			{UserID: secondReviewerID, Username: "Reviewer 2", IsActive: true},
			{UserID: inactiveUserID, Username: "Inactive User", IsActive: false},
		},
	}
)

func TestCreatePrWithTwoAssignedReviewers(t *testing.T) {
	e := setup()
	err := e.teamService.CreateTeam(e.ctx, testTeam)
	require.NoError(t, err)

	pr, err := e.prService.CreatePR(e.ctx, "pr-1", "Test PR", authorID)
	require.NoError(t, err)
	assert.Equal(t, domain.PullRequestID("pr-1"), pr.ID)
	assert.Equal(t, authorID, pr.AuthorID)

	assert.Len(t, pr.AssignedReviewers, 2)
	assert.NotContains(t, pr.AssignedReviewers, authorID)
	assert.NotContains(t, pr.AssignedReviewers, inactiveUserID)

	dbPR, exists := e.storage.PRs["pr-1"]
	require.True(t, exists)
	assert.Equal(t, "Test PR", dbPR.Name)
}

func TestCreatePRWithOneReviewer(t *testing.T) {
	e := setup()
	teamOneCandidate := domain.Team{
		Name: "one-reviewer-team",
		Members: []domain.TeamMember{
			{UserID: "author", Username: "Author", IsActive: true},
			{UserID: "rev1", Username: "Reviewer 1", IsActive: true},
			{UserID: "rev2", Username: "Reviewer 2", IsActive: false},
		},
	}
	err := e.teamService.CreateTeam(e.ctx, teamOneCandidate)
	require.NoError(t, err)

	pr, err := e.prService.CreatePR(e.ctx, "pr-1", "Test PR", "author")

	require.NoError(t, err)
	assert.Len(t, pr.AssignedReviewers, 1)
	assert.Equal(t, domain.UserID("rev1"), pr.AssignedReviewers[0])
}

func TestFailOnAlreadyExistingPR(t *testing.T) {
	e := setup()
	err := e.teamService.CreateTeam(e.ctx, testTeam)
	require.NoError(t, err)

	_, err = e.prService.CreatePR(e.ctx, "pr-1", "Test PR 1", authorID)
	require.NoError(t, err)

	_, err = e.prService.CreatePR(e.ctx, "pr-1", "Test PR 2", authorID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrPRExists)
}

func TestFailOnAuthorNotFound(t *testing.T) {
	t.Parallel()
	h := setup()

	_, err := h.prService.CreatePR(h.ctx, "pr-1", "Test PR", "non-existent-author")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func setupReassignTest(t *testing.T) (testPREnviroment, domain.PullRequest) {
	h := setup()
	err := h.teamService.CreateTeam(h.ctx, testTeam)
	require.NoError(t, err)

	pr := domain.PullRequest{
		ID: "pr-1", Name: "Test PR", AuthorID: authorID, Status: domain.StatusOpen,
		AssignedReviewers: []domain.UserID{firstReviewerID},
		CreatedAt:         time.Now(),
	}

	h.storage.PRs[pr.ID] = pr

	return h, pr
}

func TestReassignToOneAvailableCandidate(t *testing.T) {
	e, pr := setupReassignTest(t)

	updatedPR, newReviewer, err := e.prService.ReassignReviewer(e.ctx, pr.ID, firstReviewerID)
	require.NoError(t, err)
	assert.Equal(t, secondReviewerID, newReviewer)
	assert.Len(t, updatedPR.AssignedReviewers, 1)
	assert.Equal(t, secondReviewerID, updatedPR.AssignedReviewers[0])

	dbPR := e.storage.PRs[pr.ID]
	assert.Equal(t, secondReviewerID, dbPR.AssignedReviewers[0])
}

func TestFailReassignWhenPRMerged(t *testing.T) {
	e, pr := setupReassignTest(t)
	mergedPR, err := e.prService.MergePR(e.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusMerged, mergedPR.Status)

	_, _, err = e.prService.ReassignReviewer(e.ctx, pr.ID, firstReviewerID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrPRMerged)
}

func TestFailIfOldReviewerIsNotAssigned(t *testing.T) {
	e, pr := setupReassignTest(t)

	_, _, err := e.prService.ReassignReviewer(e.ctx, pr.ID, secondReviewerID)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotAssigned)
}

func TestFailWhenNoCandidateAvailable(t *testing.T) {
	e, pr := setupReassignTest(t)

	_, err := e.userRepo.SetIsActiveByID(e.ctx, secondReviewerID, false)
	require.NoError(t, err)

	_, _, err = e.prService.ReassignReviewer(e.ctx, pr.ID, firstReviewerID)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNoCandidate)
}

func TestFailReassignWhenPRNotFound(t *testing.T) {
	e := setup()

	_, _, err := e.prService.ReassignReviewer(e.ctx, "pr-not-exist", "u-1")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSuccessMergePR(t *testing.T) {
	e := setup()
	err := e.teamService.CreateTeam(e.ctx, testTeam)
	require.NoError(t, err)

	pr, err := e.prService.CreatePR(e.ctx, "pr-1", "Test PR", authorID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusOpen, pr.Status)

	mergedPR, err := e.prService.MergePR(e.ctx, "pr-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusMerged, mergedPR.Status)
	assert.NotNil(t, mergedPR.MergedAt)

	dbPR := e.storage.PRs["pr-1"]
	assert.Equal(t, domain.StatusMerged, dbPR.Status)
}

func TestFailMergeWhenPRNotFound(t *testing.T) {
	e := setup()
	err := e.teamService.CreateTeam(e.ctx, testTeam)
	require.NoError(t, err)

	mergedPR, err := e.prService.MergePR(e.ctx, "pr-1")
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Equal(t, domain.PullRequest{}, mergedPR)
}

func TestMergeIsIdempotent(t *testing.T) {
	e := setup()
	err := e.teamService.CreateTeam(e.ctx, testTeam)
	require.NoError(t, err)

	_, err = e.prService.CreatePR(e.ctx, "pr-1", "Test PR", authorID)
	require.NoError(t, err)

	mergedPR, err := e.prService.MergePR(e.ctx, "pr-1")
	require.NoError(t, err)
	firstMergeTime := mergedPR.MergedAt

	mergedPR2, err := e.prService.MergePR(e.ctx, "pr-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusMerged, mergedPR2.Status)
	assert.Equal(t, firstMergeTime, mergedPR2.MergedAt)
}
