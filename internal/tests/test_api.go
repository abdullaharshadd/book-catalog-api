package tests

// This file replaces the pytest suite tests/test_api.py, which exercised the
// Book Catalog API's root/health endpoints and full CRUD workflow against an
// isolated in-memory database with dependency-injection overrides.
//
// MIGRATION_NOTE: In Go, tests live in *_test.go files and are run by `go test`.
// The pytest class grouping (TestRootEndpoint / TestHealthEndpoint /
// TestBooksAPI) and function-scoped fixtures (test_db, client) map onto
// table-driven / per-test helper calls. The actual runnable tests are in the
// sibling file test_api_test.go so that `go test` picks them up; this file
// carries the shared HTTP-integration helpers and documentation.
//
// MIGRATION_NOTE: FastAPI's dependency_overrides[get_sync_db] mechanism has no
// direct Go analogue. Instead each test spins up a real chi router backed by an
// isolated Postgres-backed test DB via internal.NewTestServer / NewTestDB
// (defined in internal/conftest.go). Those helpers own schema creation and
// teardown, replacing SQLAlchemy's Base.metadata.create_all(bind=engine).

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// apiClient is a thin wrapper over httptest.Server that mirrors the ergonomics
// of FastAPI's TestClient (client.get / client.post / client.put /
// client.delete). It is the Go equivalent of the pytest `client` fixture.
type apiClient struct {
	t      *testing.T
	server *httptest.Server
}

// newAPIClient builds an isolated test server (fresh schema + fresh DB) and
// returns a client bound to it, registering cleanup on the test.
//
// MIGRATION_NOTE: This replaces the combined test_db + client pytest fixtures.
// internal.NewTestServer is expected to return an *httptest.Server (or an
// http.Handler wrapped in one) plus a cleanup func; adapt the exact call to the
// signature defined in internal/conftest.go.
func newAPIClient(t *testing.T) *apiClient {
	t.Helper()
	srv, cleanup := mustNewTestServer(t)
	t.Cleanup(cleanup)
	return &apiClient{t: t, server: srv}
}

// get performs a GET request and returns the response.
func (c *apiClient) get(path string) *response {
	c.t.Helper()
	return c.do(http.MethodGet, path, nil)
}

// post performs a POST request with a JSON body.
func (c *apiClient) post(path string, body any) *response {
	c.t.Helper()
	return c.do(http.MethodPost, path, body)
}

// put performs a PUT request with a JSON body.
func (c *apiClient) put(path string, body any) *response {
	c.t.Helper()
	return c.do(http.MethodPut, path, body)
}

// del performs a DELETE request.
func (c *apiClient) del(path string) *response {
	c.t.Helper()
	return c.do(http.MethodDelete, path, nil)
}

func (c *apiClient) do(method, path string, body any) *response {
	c.t.Helper()

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.server.URL+path, reader)
	if err != nil {
		c.t.Fatalf("build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.server.Client().Do(req)
	if err != nil {
		c.t.Fatalf("perform request %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatalf("read response body: %v", err)
	}

	return &response{t: c.t, statusCode: resp.StatusCode, body: raw}
}

// response is the Go equivalent of a requests/httpx Response object as used by
// FastAPI's TestClient in the source tests.
type response struct {
	t          *testing.T
	statusCode int
	body       []byte
}

// jsonMap decodes the response body into a generic map (client.json() on an
// object payload).
func (r *response) jsonMap() map[string]any {
	r.t.Helper()
	var m map[string]any
	if len(r.body) == 0 {
		return m
	}
	if err := json.Unmarshal(r.body, &m); err != nil {
		r.t.Fatalf("decode json object: %v (body=%s)", err, string(r.body))
	}
	return m
}

// jsonList decodes the response body into a slice of maps (client.json() on a
// list payload).
func (r *response) jsonList() []map[string]any {
	r.t.Helper()
	var l []map[string]any
	if len(r.body) == 0 {
		return l
	}
	if err := json.Unmarshal(r.body, &l); err != nil {
		r.t.Fatalf("decode json list: %v (body=%s)", err, string(r.body))
	}
	return l
}

// detail extracts the "detail" error message from an error response, matching
// FastAPI's error envelope shape used throughout the source assertions.
func (r *response) detail() string {
	r.t.Helper()
	m := r.jsonMap()
	if d, ok := m["detail"].(string); ok {
		return d
	}
	return ""
}
