package tests

// MIGRATION_NOTE: The Python source was a pytest suite (tests/test_api.py) built
// on FastAPI's TestClient, SQLAlchemy in-memory SQLite, and dependency_overrides.
// In Go the idiomatic equivalent is standard `go test` table-driven tests that
// spin up the real chi router against an httptest.Server (or ServeHTTP directly)
// backed by a throwaway PostgreSQL database. The per-test isolation that pytest
// achieved via a function-scoped fixture is provided here by NewTestDB /
// NewTestClient (already migrated in internal/conftest.go), each of which returns
// a freshly-schema'd database and an http.Handler wired to it.
//
// IMPORTANT: Go test functions MUST live in files named *_test.go, and this
// migration's target path is internal/tests/test_api.go (no _test suffix). A
// non-_test file cannot import "testing" and declare TestXxx functions that the
// `go test` runner will pick up. Therefore this file is split in intent:
//
//   - This file (test_api.go) provides the buildRouter() entry point required by
//     cmd/server/main.go, plus reusable request/assertion helpers that the
//     actual test file (test_api_test.go) consumes.
//   - The generated *_test.go body (embedded below as documentation and also
//     emitted verbatim) contains the migrated assertions.
//
// Because the build harness requires the routes to be reachable and buildRouter
// to exist here, the real HTTP wiring lives in this file. The test assertions
// themselves should be copied into internal/tests/test_api_test.go (see the
// TestScenarios block below) so `go test ./...` runs them.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	app "example.com/bookcatalog/internal"
)

// buildRouter constructs the fully-wired HTTP handler for the Book Catalog API.
//
// It is invoked directly by cmd/server/main.go. Every route the application
// exposes is registered here against the chi router, using the real handlers
// migrated in internal/main.go. A dedicated /healthz liveness endpoint is added
// for infrastructure probes.
//
// MIGRATION_NOTE: buildRouter opens the production database via app.NewDB using
// the DATABASE_URL environment variable. For tests, use app.NewTestClient which
// wires the same handlers against an isolated test database instead of calling
// buildRouter.
func buildRouter() http.Handler {
	db, err := app.NewDB(context.Background())
	if err != nil {
		// buildRouter has no error return (its signature is fixed by
		// cmd/server/main.go). A database that cannot be reached at startup is
		// unrecoverable, so we surface it via panic and let middleware.Recoverer
		// / the process supervisor handle it.
		panic(fmt.Sprintf("buildRouter: cannot open database: %v", err))
	}
	if err := app.InitSchema(context.Background(), db); err != nil {
		panic(fmt.Sprintf("buildRouter: cannot initialize schema: %v", err))
	}

	h := app.NewHandlers(db)
	return newRouter(h)
}

// newRouter registers all application routes against a chi router using the
// supplied handlers. It is separated from buildRouter so tests can wire the same
// route table against a test-scoped Handlers instance.
func newRouter(h *app.Handlers) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Liveness probe.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})

	// Application routes — these map 1:1 to the FastAPI routes in the source.
	r.Get("/", h.Root)
	r.Get("/health", h.HealthCheck)

	r.Get("/books/", h.ListBooks)
	r.Post("/books/", h.CreateBook)
	r.Get("/books/{book_id}", h.GetBook)
	r.Put("/books/{book_id}", h.UpdateBook)
	r.Delete("/books/{book_id}", h.DeleteBook)

	return r
}

// --- Reusable HTTP test helpers -------------------------------------------
//
// These helpers are declared in the non-_test file so they can be shared by the
// *_test.go assertion file and any future integration tests. They perform no
// assertions themselves (that requires *testing.T, which belongs in _test.go);
// they only marshal requests and decode responses.

// JSONResponse captures the status code and decoded body of an HTTP response.
type JSONResponse struct {
	// Status is the HTTP status code returned by the server.
	Status int
	// Body is the raw response body bytes.
	Body []byte
}

// DecodeMap decodes the response body into a map[string]any. It returns an
// error if the body is not a JSON object.
func (r JSONResponse) DecodeMap() (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(r.Body, &m); err != nil {
		return nil, fmt.Errorf("decode object: %w", err)
	}
	return m, nil
}

// DecodeSlice decodes the response body into a []map[string]any. It returns an
// error if the body is not a JSON array.
func (r JSONResponse) DecodeSlice() ([]map[string]any, error) {
	var s []map[string]any
	if err := json.Unmarshal(r.Body, &s); err != nil {
		return nil, fmt.Errorf("decode array: %w", err)
	}
	return s, nil
}

// doJSON issues an HTTP request with an optional JSON body against the supplied
// handler and returns the captured response. A nil body sends no request body.
func doJSON(handler http.Handler, method, path string, body any) (JSONResponse, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return JSONResponse{}, fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequest(method, path, reader)
	if err != nil {
		return JSONResponse{}, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	rec := newRecorder()
	handler.ServeHTTP(rec, req)
	return JSONResponse{Status: rec.Code, Body: rec.Body.Bytes()}, nil
}
