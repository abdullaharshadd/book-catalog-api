package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"migrated-app/internal/database"
	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
	"migrated-app/internal/main"
	"migrated-app/internal/tests/conftest.go"
)

const (
	rootURL = "/"
	healthURL = "/health"
	booksURL = "/books/"
	docsURL = "/docs"
	welcomeMessage = "Welcome to Book Catalog API"
	version = "1.0.0"
	healthyStatus = "healthy"
	serviceName = "book-catalog-api"
	invalidYear = 999
)

// TestClient returns a new test HTTP client.
func TestClient(t *testing.T) *httptest.Client {
	t.Helper()

	server := httptest.NewServer(main.buildRouter())
	return server.Client()
}

// TestRootEndpoint tests the root endpoint.
func TestRootEndpoint(t *testing.T) {
	client := TestClient(t)
	response := client.Get(rootURL)
	if response.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, response.StatusCode)
	}
	body, _ := ioutil.ReadAll(response.Body)
	defer response.Body.Close()

	var data map[string]interface{}
	json.Unmarshal(body, &data)

	if data["message"] != welcomeMessage {
		t.Errorf("expected message %s, got %s", welcomeMessage, data["message"])
	}
	if data["version"] != version {
		t.Errorf("expected version %s, got %s", version, data["version"])
	}
	if _, ok := data["docs_url"]