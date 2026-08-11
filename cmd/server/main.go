package main

import (
	"net/http"
	"os"

	"github.com/rs/zerolog/log"

	"book_catalog_api/internal"
)

func main() {
	db, err := internal.NewDB()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	router := internal.BuildRouter(db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Info().Str("port", port).Msg("starting server")
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}