package postgres

import (
	"context"
	"errors"
	"pr-reviewer-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

func (ur *UserRepo) Create(ctx context.Context, user domain.User) error {
	createUserQuery := `
		INSERT INTO users (user_id, username, team_name, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET
			username = EXCLUDED.username,
			team_name = EXCLUDED.team_name,
			is_active = EXCLUDED.is_active
	`

	_, err := ur.db.Exec(ctx, createUserQuery, user.ID, user.Username, user.TeamName, user.IsActive)
	return err
}

func (ur *UserRepo) UserByID(ctx context.Context, userID domain.UserID) (domain.User, error) {
	userByIDQuery := `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE user_id = $1
	`

	var user domain.User
	err := ur.db.QueryRow(ctx, userByIDQuery, userID).
		Scan(&user.ID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}

	return user, nil
}

func (ur *UserRepo) SetIsActiveByID(ctx context.Context, userID domain.UserID, isActive bool) (domain.User, error) {
	setIsActiveQuery := `
		UPDATE users
		SET is_active = $2
		WHERE user_id = $1
		RETURNING user_id, username, team_name, is_active
	`

	var user domain.User
	err := ur.db.QueryRow(ctx, setIsActiveQuery, userID, isActive).
		Scan(&user.ID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}

	return user, nil
}

func (ur *UserRepo) ActiveUsersByTeamName(ctx context.Context, teamName domain.TeamName) ([]domain.User, error) {
	activeUsersQuery := `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE team_name = $1 AND is_active = TRUE
	`

	rows, err := ur.db.Query(ctx, activeUsersQuery, teamName)
	if err != nil {
		return []domain.User{}, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.ID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return []domain.User{}, err
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return []domain.User{}, err
	}

	return users, nil
}
