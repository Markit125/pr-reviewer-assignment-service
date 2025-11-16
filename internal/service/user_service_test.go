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

type testUserEnviroment struct {
	ctx     context.Context
	storage *inmemory.InMemoryStorage

	userRepo repository.UserRepository
	teamRepo repository.TeamRepository
	prRepo   repository.PullRequestRepository

	userService *service.UserService
}

func setupUserTest() testUserEnviroment {
	storage, _ := inmemory.NewStorage()

	userRepo := inmemory.NewUserRepo(storage)
	teamRepo := inmemory.NewTeamRepo(storage)
	prRepo := inmemory.NewPullRequestRepo(storage)

	userService := service.NewUserService(userRepo, prRepo)

	return testUserEnviroment{
		ctx:         context.Background(),
		storage:     storage,
		userRepo:    userRepo,
		teamRepo:    teamRepo,
		prRepo:      prRepo,
		userService: userService,
	}
}

var (
	userTeamName = domain.TeamName("test-team")
	userID1      = domain.UserID("u-1")
	userID2      = domain.UserID("u-2")
	prID1        = domain.PullRequestID("pr-1")
	prID2        = domain.PullRequestID("pr-2")
	prID3        = domain.PullRequestID("pr-3")

	testUser1 = domain.User{
		ID:       userID1,
		Username: "User One",
		TeamName: userTeamName,
		IsActive: true,
	}
	testUser2 = domain.User{
		ID:       userID2,
		Username: "User Two",
		TeamName: userTeamName,
		IsActive: true,
	}
	testPR1 = domain.PullRequest{
		ID:                prID1,
		Name:              "PR 1",
		AuthorID:          userID2,
		Status:            domain.StatusOpen,
		AssignedReviewers: []domain.UserID{userID1},
		CreatedAt:         time.Now(),
	}
	testPR2 = domain.PullRequest{
		ID:                prID2,
		Name:              "PR 2",
		AuthorID:          userID2,
		Status:            domain.StatusMerged,
		AssignedReviewers: []domain.UserID{userID1},
		CreatedAt:         time.Now().Add(1 * time.Minute),
	}
	testPR3 = domain.PullRequest{
		ID:                prID3,
		Name:              "PR 3",
		AuthorID:          userID1,
		Status:            domain.StatusOpen,
		AssignedReviewers: []domain.UserID{userID2},
		CreatedAt:         time.Now().Add(2 * time.Minute),
	}
)

func TestSetUserActivityDeactivatesUser(t *testing.T) {
	e := setupUserTest()
	e.storage.Users[userID1] = testUser1
	require.True(t, e.storage.Users[userID1].IsActive)

	updatedUser, err := e.userService.SetIsActive(e.ctx, userID1, false)

	require.NoError(t, err)
	assert.False(t, updatedUser.IsActive)
	assert.Equal(t, userID1, updatedUser.ID)

	dbUser := e.storage.Users[userID1]
	assert.False(t, dbUser.IsActive)
}

func TestSetUserActivityActivatesUser(t *testing.T) {
	e := setupUserTest()
	inactiveUser := testUser1
	inactiveUser.IsActive = false
	e.storage.Users[userID1] = inactiveUser
	require.False(t, e.storage.Users[userID1].IsActive)

	updatedUser, err := e.userService.SetIsActive(e.ctx, userID1, true)

	require.NoError(t, err)
	assert.True(t, updatedUser.IsActive)
	assert.Equal(t, userID1, updatedUser.ID)

	dbUser := e.storage.Users[userID1]
	assert.True(t, dbUser.IsActive)
}

func TestSetUserActivityFailsOnNotFound(t *testing.T) {
	e := setupUserTest()

	_, err := e.userService.SetIsActive(e.ctx, "non-existent-user", true)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func setupReviewTest(t *testing.T) testUserEnviroment {
	e := setupUserTest()
	e.storage.Users[userID1] = testUser1
	e.storage.Users[userID2] = testUser2
	e.storage.PRs[prID1] = testPR1
	e.storage.PRs[prID2] = testPR2
	e.storage.PRs[prID3] = testPR3
	return e
}

func TestReviewAssignmentsSuccess(t *testing.T) {
	e := setupReviewTest(t)

	assignments, err := e.userService.ReviewAssignments(e.ctx, userID1)

	require.NoError(t, err)
	assert.Equal(t, userID1, assignments.UserID)
	require.Len(t, assignments.PullRequests, 2)

	prIDs := []domain.PullRequestID{
		assignments.PullRequests[0].ID,
		assignments.PullRequests[1].ID,
	}
	assert.Contains(t, prIDs, prID1)
	assert.Contains(t, prIDs, prID2)
	assert.NotContains(t, prIDs, prID3)
}

func TestGetReviewAssignmentsZeroPRs(t *testing.T) {
	e := setupUserTest()

	e.storage.Users[userID1] = testUser1

	assignments, err := e.userService.ReviewAssignments(e.ctx, userID1)
	require.NoError(t, err)
	assert.Equal(t, userID1, assignments.UserID)
	assert.Len(t, assignments.PullRequests, 0)
}

func TestGetReviewAssignmentsFailsOnNotFound(t *testing.T) {
	e := setupReviewTest(t)

	_, err := e.userService.ReviewAssignments(e.ctx, "u-ghost")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
