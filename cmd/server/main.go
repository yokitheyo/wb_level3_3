package main

import (
	"log"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"

	httpHandler "github.com/yokitheyo/wb_level3_3/internal/handler/http"
	"github.com/yokitheyo/wb_level3_3/internal/repository/postgres"
	"github.com/yokitheyo/wb_level3_3/internal/usecase"
)

func main() {
	// --- конфиг (замени на env/flags)
	masterDSN := "postgres://postgres:postgres@localhost:5432/commenttree?sslmode=disable"
	slaves := []string{}
	opts := &dbpg.Options{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
	}

	// init wbf logger
	if err := zlog.New(); err != nil {
		log.Fatalf("zlog.New: %v", err)
	}

	// init dbpg (внутри dbpg происходит импорт драйвера pq)
	db, err := dbpg.New(masterDSN, slaves, opts)
	if err != nil {
		zlog.Error().Err(err).Msg("dbpg.New failed")
		log.Fatalf("dbpg.New: %v", err)
	}

	// run goose migrations using db.Master (driver уже зарегистрирован через dbpg)
	goose.SetDialect("postgres")
	if err := goose.Up(db.Master, "./migrations"); err != nil {
		zlog.Error().Err(err).Msg("goose.Up failed")
		log.Fatalf("goose up: %v", err)
	}
	zlog.Info().Msg("migrations applied")

	// repo / usecase / handlers
	repo := postgres.NewCommentRepository(db)
	uc := usecase.NewCommentUsecase(repo)
	handler := httpHandler.NewCommentHandler(uc)

	// router (wbf/ginext)
	r := ginext.New()
	r.Use(ginext.Logger(), ginext.Recovery())

	httpHandler.RegisterWeb(r)
	handler.RegisterRoutes(r)

	zlog.Info().Msg("server start :8080")
	if err := r.Run(":8080"); err != nil {
		zlog.Error().Err(err).Msg("server stopped")
		log.Fatalf("server stopped: %v", err)
	}
}
