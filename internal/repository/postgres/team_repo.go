package postgres

import (
	"context"
	"errors"
	"pr-reviewer-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeamRepo struct {
	db *pgxpool.Pool
}

func NewTeamRepo(db *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{
		db: db,
	}
}

func (tr *TeamRepo) Create(ctx context.Context, team domain.Team) error {
	tx, err := tr.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	createTeamQuery := `INSERT INTO teams (team_name) VALUES ($1)`
	if _, err := tx.Exec(ctx, createTeamQuery, team.Name); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrTeamExists
		}

		return err
	}

	createUserQuery := `
		INSERT INTO users(user_id, username, team_name, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET
			username = EXCLUDED.username,
			team_name = EXCLUDED.team_name,
			is_active = EXCLUDED.is_active
	`

	batch := &pgx.Batch{}
	for _, member := range team.Members {
		batch.Queue(createUserQuery, member.UserID, member.Username, team.Name, member.IsActive)
	}

	batchRes := tx.SendBatch(ctx, batch)
	if err := batchRes.Close(); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (tr *TeamRepo) TeamByName(ctx context.Context, teamName domain.TeamName) (domain.Team, error) {
	teamQuery := `
		SELECT t.team_name, u.user_id, u.user_name, u.is_active
		FROM teams t
		LEFT JOIN users u ON t.team_name = u.team_name
		WHERE t.team_name = $1
	`

	rows, err := tr.db.Query(ctx, teamQuery, teamName)
	if err != nil {
		return domain.Team{}, err
	}
	defer rows.Close()

	team := domain.Team{}
	members := []domain.TeamMember{}

	for rows.Next() {
		var (
			member   domain.TeamMember
			tn       domain.TeamName
			uid      domain.UserID
			username string
			isActive bool
		)

		if err := rows.Scan(&tn, &uid, &username, &isActive); err != nil {
			return domain.Team{}, err
		}

		if team.Name == "" {
			team.Name = tn
		}

		if uid != "" {
			member.UserID = uid
			member.Username = username
			member.IsActive = isActive
			members = append(members, member)
		}
	}

	if rows.Err() != nil {
		return domain.Team{}, err
	}

	if team.Name == "" {
		return domain.Team{}, domain.ErrNotFound
	}

	team.Members = members

	return team, nil
}
