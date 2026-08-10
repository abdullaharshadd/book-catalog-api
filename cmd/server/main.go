package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/book-catalog-api/internal"
)

func main() {
	db, err := internal.OpenDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(context.Background()); err != nil {
		log.Fatalf("failed to initialize schema: %v", err)
	}

	router := internal.BuildRouter(db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}