package inmemory

import (
	"context"
	"pr-reviewer-service/internal/domain"
)

type UserRepo struct {
	db *InMemoryStorage
}

func NewUserRepo(db *InMemoryStorage) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

func (ur *UserRepo) Create(ctx context.Context, user domain.User) error {
	ur.db.Users[user.ID] = user
	return nil
}

func (ur *UserRepo) UserByID(ctx context.Context, userID domain.UserID) (domain.User, error) {
	user, exists := ur.db.Users[userID]
	if !exists {
		return domain.User{}, domain.ErrNotFound
	}

	return user, nil
}

func (ur *UserRepo) SetIsActiveByID(_ context.Context, userID domain.UserID, isActive bool) (domain.User, error) {
	user, exists := ur.db.Users[userID]
	if !exists {
		return domain.User{}, domain.ErrNotFound
	}

	user.IsActive = isActive
	ur.db.Users[userID] = user

	return user, nil
}

func (ur *UserRepo) ActiveUsersByTeamName(_ context.Context, teamName domain.TeamName) ([]domain.User, error) {
	users := []domain.User{}

	for _, member := range ur.db.Users {
		if member.TeamName == teamName && member.IsActive {
			users = append(users, member)
		}
	}

	return users, nil
}
