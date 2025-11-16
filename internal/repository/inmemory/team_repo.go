package inmemory

import (
	"context"
	"pr-reviewer-service/internal/domain"
)

type TeamRepo struct {
	db *InMemoryStorage
}

func NewTeamRepo(db *InMemoryStorage) *TeamRepo {
	return &TeamRepo{
		db: db,
	}
}

func (tr *TeamRepo) Create(_ context.Context, team domain.Team) error {
	if _, exists := tr.db.Teams[team.Name]; exists {
		return domain.ErrTeamExists
	}

	for _, member := range team.Members {
		tr.db.Users[member.UserID] = domain.User{
			ID:       member.UserID,
			Username: member.Username,
			TeamName: team.Name,
			IsActive: member.IsActive,
		}
	}

	tr.db.Teams[team.Name] = team

	return nil
}

func (tr *TeamRepo) TeamByName(ctx context.Context, teamName domain.TeamName) (domain.Team, error) {
	if team, exists := tr.db.Teams[teamName]; exists {
		members := []domain.TeamMember{}
		for _, member := range tr.db.Users {
			if member.TeamName == teamName {
				members = append(members, domain.TeamMember{
					UserID:   member.ID,
					Username: member.Username,
					IsActive: member.IsActive,
				})
			}
		}

		team.Members = members

		return team, nil
	}

	return domain.Team{}, domain.ErrNotFound
}
