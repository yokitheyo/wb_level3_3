package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/wb_level3_3/internal/handler/middleware"
	infradatabase "github.com/yokitheyo/wb_level3_3/internal/infrastructure/database"
	"github.com/yokitheyo/wb_level3_3/internal/infrastructure/search"
	"github.com/yokitheyo/wb_level3_3/internal/retry"

	"github.com/yokitheyo/wb_level3_3/internal/config"
	httpHandler "github.com/yokitheyo/wb_level3_3/internal/handler/http"
	"github.com/yokitheyo/wb_level3_3/internal/repository/postgres"
	"github.com/yokitheyo/wb_level3_3/internal/usecase"
)

// splitAndTrim splits s by sep and trims empty parts.
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func main() {
	zlog.Init()
	zlog.Logger.Info().Msg("zlog initialized")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load config
	configPath := "config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "/app/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to load config")
	}

	// Setup DB
	masterDSN := cfg.Database.DSN
	slaves := []string{}
	if strings.TrimSpace(cfg.Database.Slaves) != "" {
		slaves = splitAndTrim(cfg.Database.Slaves, ",")
	}
	dbOpts := &dbpg.Options{
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: time.Duration(cfg.Database.ConnMaxLifetimeSec) * time.Second,
	}

	var database *dbpg.DB
	for i := 0; i < cfg.Database.ConnectRetries; i++ {
		database, err = dbpg.New(masterDSN, slaves, dbOpts)
		if err == nil {
			if pingErr := database.Master.Ping(); pingErr == nil {
				break
			} else {
				zlog.Logger.Warn().Err(pingErr).Msg("db ping failed")
				err = pingErr
			}
		}
		zlog.Logger.Warn().Err(err).Msgf("waiting for database (attempt %d/%d)", i+1, cfg.Database.ConnectRetries)
		time.Sleep(time.Duration(cfg.Database.ConnectRetryDelaySec) * time.Second)
	}
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to connect to database after retries")
	}

	// Run migrations
	if err := infradatabase.RunMigrations(database, cfg.Migrations.Path); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("migrations failed")
	}

	// Setup repository and usecase
	repo := postgres.NewCommentRepository(database, retry.DefaultStrategy)

	// Full-text search adapter
	fts := search.NewPostgresFullText(repo)

	// Setup usecase с search
	uc := usecase.NewCommentUsecase(repo, fts)

	// Setup Gin engine + handlers
	engine := ginext.New()
	engine.Use(middleware.LoggerMiddleware(), middleware.CORSMiddleware())

	engine.Static("/", "./static")

	commentHandler := httpHandler.NewCommentHandler(uc)
	commentHandler.RegisterRoutes(engine)

	// Start HTTP server
	srv := &http.Server{
		Addr:    cfg.Server.Addr,
		Handler: engine,
	}

	go func() {
		zlog.Logger.Info().Str("addr", cfg.Server.Addr).Msg("starting HTTP server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zlog.Logger.Fatal().Err(err).Msg("failed to start API server")
		}
	}()

	// Graceful shutdown
	<-ctx.Done()
	zlog.Logger.Info().Msg("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeoutSec)*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		zlog.Logger.Error().Err(err).Msg("HTTP server shutdown failed")
	} else {
		zlog.Logger.Info().Msg("HTTP server stopped gracefully")
	}

	if err := database.Master.Close(); err != nil {
		zlog.Logger.Error().Err(err).Msg("closing db master failed")
	} else {
		zlog.Logger.Info().Msg("db master closed")
	}
	for i, s := range database.Slaves {
		if err := s.Close(); err != nil {
			zlog.Logger.Error().Err(err).Int("slave_index", i).Msg("closing db slave failed")
		}
	}

	zlog.Logger.Info().Msg("shutdown complete")
}
