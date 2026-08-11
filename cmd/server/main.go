package main

import (
	"log"
	"net/http"
	"os"

	"github.com/migrated-app/internal"
)

func main() {
	db := internal.ConnectDB()

	internal.InitServer(db)

	router := internal.BuildRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}