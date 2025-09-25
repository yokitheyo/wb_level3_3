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

	zlog.Logger.Info().Msgf("Loading config from: %s", configPath)

	cfg, err := config.Load(configPath)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to load config")
	}
	fmt.Printf("%+v\n", cfg.Database)

	zlog.Logger.Info().Msgf("DSN from config: %s", cfg.Database.DSN)
	zlog.Logger.Info().Msgf("Config retries: %d, delay: %d", cfg.Database.ConnectRetries, cfg.Database.ConnectRetryDelaySec)

	// ВРЕМЕННО: Принудительно устанавливаем значения если они 0
	connectRetries := cfg.Database.ConnectRetries
	connectDelay := cfg.Database.ConnectRetryDelaySec

	if connectRetries == 0 {
		connectRetries = 15 // хардкод
		zlog.Logger.Warn().Msg("connect_retries was 0, using hardcoded value 15")
	}
	if connectDelay == 0 {
		connectDelay = 3 // хардкод
		zlog.Logger.Warn().Msg("connect_retry_delay_sec was 0, using hardcoded value 3")
	}

	// Setup DB with more detailed logging
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

	zlog.Logger.Info().Msgf("Attempting to connect to database with %d retries, %d second delay",
		connectRetries, connectDelay)

	var database *dbpg.DB
	for i := 0; i < connectRetries; i++ {
		zlog.Logger.Info().Msgf("Database connection attempt %d/%d", i+1, connectRetries)

		database, err = dbpg.New(masterDSN, slaves, dbOpts)
		if err != nil {
			zlog.Logger.Warn().Err(err).Msgf("dbpg.New failed on attempt %d/%d", i+1, connectRetries)
			database = nil
		} else {
			zlog.Logger.Info().Msg("dbpg.DB created, testing connection...")

			if database.Master == nil {
				err = fmt.Errorf("database.Master is nil")
				zlog.Logger.Warn().Err(err).Msgf("nil master connection on attempt %d/%d", i+1, connectRetries)
			} else if pingErr := database.Master.Ping(); pingErr != nil {
				zlog.Logger.Warn().Err(pingErr).Msgf("db ping failed on attempt %d/%d", i+1, connectRetries)
				err = pingErr
				if database.Master != nil {
					database.Master.Close()
				}
				for _, slave := range database.Slaves {
					if slave != nil {
						slave.Close()
					}
				}
				database = nil
			} else {
				zlog.Logger.Info().Msg("Database connection established successfully")
				break
			}
		}

		if i < connectRetries-1 {
			zlog.Logger.Info().Msgf("Waiting %d seconds before next attempt...", connectDelay)
			time.Sleep(time.Duration(connectDelay) * time.Second)
		}
	}

	if err != nil || database == nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to connect to database after all retries")
	}

	zlog.Logger.Info().Msg("Database connection successful, running migrations...")

	// Run migrations
	if err := infradatabase.RunMigrations(database, cfg.Migrations.Path); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("migrations failed")
	}

	zlog.Logger.Info().Msg("Migrations completed successfully")

	// Setup repository and usecase
	repo := postgres.NewCommentRepository(database, retry.DefaultStrategy)

	// Full-text search adapter
	fts := search.NewPostgresFullText(repo)

	// Setup usecase с search
	uc := usecase.NewCommentUsecase(repo, fts)

	// Setup Gin engine + handlers
	engine := ginext.New()
	engine.Use(middleware.LoggerMiddleware(), middleware.CORSMiddleware())

	engine.GET("/", func(c *ginext.Context) {
		c.File("./static/index.html")
	})
	engine.Static("/static", "./static")

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

	if database != nil && database.Master != nil {
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
	}

	zlog.Logger.Info().Msg("shutdown complete")
}
