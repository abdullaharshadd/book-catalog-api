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
	"migrated-app/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
	
	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database")
	}
	defer db.Close()
	
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: main.buildRouter(),
	}
	
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()
	
	log.Info().Msg("server started on :" + cfg.Port)
	<-ctx.Done()
	
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	delayCancel(cancel)
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
	}
}

func delayCancel(cancel context.CancelFunc) {
	time.AfterFunc(10*time.Second, cancel)
}