package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"book-catalog-api/internal"
)

func main() {
	db, err := internal.OpenDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := internal.InitSchema(db); err != nil {
		log.Fatalf("failed to initialize schema: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	router := internal.BuildRouter(db)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}