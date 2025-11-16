package service_test

import (
	"context"
	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/repository/inmemory"
	"pr-reviewer-service/internal/service"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testTeamEnviroment struct {
	ctx     context.Context
	storage *inmemory.InMemoryStorage

	userRepo repository.UserRepository
	teamRepo repository.TeamRepository

	teamService *service.TeamService
}

func setupTeamTest() testTeamEnviroment {
	storage, _ := inmemory.NewStorage()

	userRepo := inmemory.NewUserRepo(storage)
	teamRepo := inmemory.NewTeamRepo(storage)

	teamService := service.NewTeamService(teamRepo, userRepo)

	return testTeamEnviroment{
		ctx:         context.Background(),
		storage:     storage,
		userRepo:    userRepo,
		teamRepo:    teamRepo,
		teamService: teamService,
	}
}

var (
	teamPlatformName = domain.TeamName("platform")
	firstUserID      = domain.UserID("first-user-id")
	secondUserID     = domain.UserID("second-user-id")

	teamPlatform = domain.Team{
		Name: teamPlatformName,
		Members: []domain.TeamMember{
			{UserID: firstUserID, Username: "First", IsActive: true},
			{UserID: secondUserID, Username: "Second", IsActive: false},
		},
	}
)

func TestSuccessCreateTeam(t *testing.T) {
	e := setupTeamTest()

	err := e.teamService.CreateTeam(e.ctx, teamPlatform)
	require.NoError(t, err)

	dbTeam, exists := e.storage.Teams[teamPlatformName]
	require.True(t, exists, "Team was not saved in the map")
	assert.Equal(t, teamPlatformName, dbTeam.Name)

	userFirst, exists := e.storage.Users[firstUserID]
	require.True(t, exists, "User 'First' was not saved")
	assert.Equal(t, "First", userFirst.Username)
	assert.Equal(t, teamPlatformName, userFirst.TeamName)
	assert.True(t, userFirst.IsActive)

	userSecond, exists := e.storage.Users[secondUserID]
	require.True(t, exists, "User 'Second' was not saved")
	assert.Equal(t, "Second", userSecond.Username)
	assert.Equal(t, teamPlatformName, userSecond.TeamName)
	assert.False(t, userSecond.IsActive)
}

func TestFailOnAlreadyExistingTeam(t *testing.T) {
	e := setupTeamTest()
	err1 := e.teamService.CreateTeam(e.ctx, teamPlatform)
	require.NoError(t, err1)

	err2 := e.teamService.CreateTeam(e.ctx, teamPlatform)
	require.Error(t, err2)
	assert.ErrorIs(t, err2, domain.ErrTeamExists)
}

func TestCreateTeamUpdatedUsers(t *testing.T) {
	e := setupTeamTest()

	frontendTeam := domain.Team{
		Name: "frontend",
		Members: []domain.TeamMember{
			{UserID: firstUserID, Username: "First-Frontend", IsActive: false},
		},
	}
	err := e.teamService.CreateTeam(e.ctx, frontendTeam)
	require.NoError(t, err)

	userFirst := e.storage.Users[firstUserID]
	require.Equal(t, domain.TeamName("frontend"), userFirst.TeamName)
	require.False(t, userFirst.IsActive)

	err = e.teamService.CreateTeam(e.ctx, teamPlatform)
	require.NoError(t, err)

	userFirstUpdated := e.storage.Users[firstUserID]
	assert.Equal(t, teamPlatformName, userFirstUpdated.TeamName)
	assert.Equal(t, "First", userFirstUpdated.Username)
	assert.True(t, userFirstUpdated.IsActive)

	_, exists := e.storage.Users[secondUserID]
	assert.True(t, exists)
}

func TestGetTeamSuccess(t *testing.T) {
	e := setupTeamTest()
	err := e.teamService.CreateTeam(e.ctx, teamPlatform)
	require.NoError(t, err)

	team, err := e.teamService.Team(e.ctx, teamPlatformName)

	require.NoError(t, err)
	assert.Equal(t, teamPlatformName, team.Name)

	assert.ElementsMatch(t, teamPlatform.Members, team.Members)
}

func TestGetTeamFailNotFound(t *testing.T) {
	e := setupTeamTest()

	_, err := e.teamService.Team(e.ctx, "non-existent-team")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
