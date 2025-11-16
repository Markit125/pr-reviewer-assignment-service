package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pr-reviewer-service/internal/config"
	"pr-reviewer-service/internal/repository/postgres"
	"pr-reviewer-service/internal/service"
	httptransport "pr-reviewer-service/internal/transport/http"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // driver
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("application startup error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.LoadConfig()

	logger.Info("connecting to database...")
	dbPool, err := postgres.NewPsqlConnection(postgres.Config{DSN: cfg.DatabaseDSN})
	if err != nil {
		return err
	}
	defer dbPool.Close()
	logger.Info("database connection established")

	logger.Info("running database migrations...")
	m, err := migrate.New("file://migrations", cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	logger.Info("database migrations complete")

	teamRepo := postgres.NewTeamRepo(dbPool)
	userRepo := postgres.NewUserRepo(dbPool)
	prRepo := postgres.NewPullRequestRepo(dbPool)

	teamService := service.NewTeamService(teamRepo, userRepo)
	userService := service.NewUserService(userRepo, prRepo)
	prService := service.NewPullRequestService(prRepo, userRepo)

	httpHandler := httptransport.NewHandler(teamService, userService, prService, logger)

	router := httpHandler.RegisterRoutes()

	srv := &http.Server{
		Addr:         cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("server starting", "port", cfg.ServerPort)
		serverErrors <- srv.ListenAndServe()
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return err
	case sig := <-stopChan:
		logger.Info("shutdown signal received", "signal", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	logger.Info("server shut down gracefully")
	return nil
}
