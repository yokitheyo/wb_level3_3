package db

import (
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/zlog"
)

func RunMigrations(db *dbpg.DB, migrationsDir string) error {
	zlog.Logger.Info().Msgf("starting migrations from: %s", migrationsDir)

	goose.SetDialect("postgres")

	if err := goose.Up(db.Master, migrationsDir); err != nil {
		zlog.Logger.Error().Err(err).Msg("failed to apply migrations")
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	zlog.Logger.Info().Msg("migrations applied successfully")
	return nil
}
