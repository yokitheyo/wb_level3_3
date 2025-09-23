package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/zlog"

	"github.com/yokitheyo/wb_level3_3/internal/config"
	"github.com/yokitheyo/wb_level3_3/internal/repository/postgres"
	"github.com/yokitheyo/wb_level3_3/internal/usecase"
)

// splitAndTrim splits s by sep and trims results.
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

	configPath := "config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "/app/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to load config")
	}

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

	goose.SetDialect("postgres")
	if err := goose.Up(database.Master, cfg.Migrations.Path); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("migrations failed")
	}
	zlog.Logger.Info().Msg("migrations applied")

	repo := postgres.NewCommentRepository(database)
	uc := usecase.NewCommentUsecase(repo)

	// optional: init redis cache if configured
	var cacheSvc *cache.RedisCache
	if cfg.Redis != nil && cfg.Redis.Addr != "" {
		rdb := redis.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
		cacheSvc = cache.NewRedisCache(rdb, cfg.Cache.Prefix, retry.DefaultStrategy)
		zlog.Logger.Info().Str("redis", cfg.Redis.Addr).Msg("redis cache initialized")
	}

	// combine into app (aggregates repo/cache/usecase)
	appSvc := app.NewApp(repo, cacheSvc, uc)

	// --- Create API server (internal/api should provide NewAPI(app) returning server with Start/Stop)
	apiServer := api.NewAPI(appSvc)

	// start API server in goroutine
	go func() {
		zlog.Logger.Info().Str("addr", cfg.Server.Addr).Msg("starting HTTP server")
		if err := apiServer.Start(cfg.Server.Addr); err != nil && err != http.ErrServerClosed {
			zlog.Logger.Fatal().Err(err).Msg("failed to start API server")
		}
	}()

	// wait for shutdown signal
	<-ctx.Done()
	zlog.Logger.Info().Msg("shutdown signal received")

	// give api server some time to stop gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeoutSec)*time.Second)
	defer cancel()

	if err := a
