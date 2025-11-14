package postgres

import "github.com/jackc/pgx/v5/pgxpool"

type PsqlConnectionStrategy func(Config) (*pgxpool.Pool, error)

type PostgresRetrier struct {
	countRetries   int
	connectionFunc PsqlConnectionStrategy
}

func NewPostgresRetrier(countRetries int, connectionFunc PsqlConnectionStrategy) *PostgresRetrier {
	return &PostgresRetrier{
		countRetries:   countRetries,
		connectionFunc: connectionFunc,
	}
}

func (r *PostgresRetrier) newConnection(cfg Config, connectionFunc PsqlConnectionStrategy) (*pgxpool.Pool, error) {
	db, err := connectionFunc(cfg)

	for err != nil && r.countRetries > 0 {
		r.countRetries--
		db, err = connectionFunc(cfg)
	}

	return db, err
}

func NewPsqlConnectionWithRetrier(cfg Config, retrier *PostgresRetrier) (*pgxpool.Pool, error) {
	return retrier.newConnection(cfg, retrier.connectionFunc)
}
