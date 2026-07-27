package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"migrated-app/internal"
	"migrated-app/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	if cfg.DatabaseURL == "" {
		// Try to build DSN from individual env vars
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = os.Getenv("POSTGRES_HOST")
		}
		if host == "" {
			host = "db"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = os.Getenv("POSTGRES_PORT")
		}
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = os.Getenv("POSTGRES_USER")
		}
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("DB_PASSWORD")
		if password == "" {
			password = os.Getenv("POSTGRES_PASSWORD")
		}
		dbname := os.Getenv("DB_NAME")
		if dbname == "" {
			dbname = os.Getenv("POSTGRES_DB")
		}
		if dbname == "" {
			dbname = "postgres"
		}
		cfg.DatabaseURL = "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + dbname + "?sslmode=disable"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var db interface{ Close() error }
	for i := 0; i < 10; i++ {
		db, err = internal.NewDB(ctx, cfg.DatabaseURL)
		if err == nil {
			break
		}
		log.Warn().Err(err).Msgf("failed to connect to database, retrying in 2s (attempt %d/10)", i+1)
		select {
		case <-ctx.Done():
			log.Fatal().Msg("context cancelled during DB connect retry")
		case <-time.After(2 * time.Second):
		}
	}
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database after retries")
	}
	defer db.Close()

	if err := internal.InitDB(ctx, db); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database schema")
	}

	repo := internal.NewBookRepository(db)
	handler := internal.NewBookHandler(repo)
	internal.SetHandler(handler)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: internal.BuildRouter(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	log.Info().Msgf("server started on :%s", cfg.Port)
	<-ctx.Done()

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
	}
}