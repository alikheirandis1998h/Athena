// Package main is the entrypoint for the API server.
//
// @title                       Base Backend API
// @version                     1.0
// @description                 Reusable Go backend template (Gin + sqlc/pgx + JWT auth).
// @termsOfService               http://swagger.io/terms/
//
// @contact.name                API Support
//
// @license.name                MIT
//
// @host                        localhost:8080
// @BasePath                    /
//
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and the access token.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	_ "base-backend/docs"
	"base-backend/internal/auth"
	"base-backend/internal/config"
	"base-backend/internal/database"
	"base-backend/internal/db"
	"base-backend/internal/handlers"
	"base-backend/internal/logger"
	"base-backend/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger.Init(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	queries := db.New(pool)
	tokenManager := auth.NewTokenManager(cfg.JWTAccessSecret, cfg.JWTRefreshSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	deps := server.Dependencies{
		TokenManager: tokenManager,
		AuthHandler:  handlers.NewAuthHandler(queries, tokenManager),
		UserHandler:  handlers.NewUserHandler(queries),
	}

	router := server.NewRouter(cfg.Env, cfg.CORSAllowedOrigins, deps)
	srv := server.New(cfg.Port, router)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
	}
}
