package config_test

import (
	"os"
	"pr-reviewer-service/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Defaults(t *testing.T) {
	originalPort := os.Getenv("PORT")
	originalDSN := os.Getenv("DATABASE_DSN")

	require.NoError(t, os.Unsetenv("PORT"))
	require.NoError(t, os.Unsetenv("DATABASE_DSN"))

	t.Cleanup(func() {
		os.Setenv("PORT", originalPort)
		os.Setenv("DATABASE_DSN", originalDSN)
	})

	cfg := config.LoadConfig()

	assert.Equal(t, ":8080", cfg.ServerPort)
	assert.Equal(t, "postgres://user:password@localhost:5432/pr_reviewer_db?sslmode=disable", cfg.DatabaseDSN)
}

func TestLoadConfig_FromEnv(t *testing.T) {
	t.Setenv("PORT", "1234")
	t.Setenv("DATABASE_DSN", "my-dsn-string")

	cfg := config.LoadConfig()

	assert.Equal(t, ":1234", cfg.ServerPort)
	assert.Equal(t, "my-dsn-string", cfg.DatabaseDSN)
}
