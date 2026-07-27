```go
package internal_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Interfaces & fakes that stand in for the real DB / app wiring
// (mirrors what internal/database.go and internal/main.go would expose)
// ---------------------------------------------------------------------------

// DBSession is the minimal interface a test session must satisfy.
type DBSession interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Close() error
}

// MockSession is an in-memory, goroutine-safe fake that records calls.
type MockSession struct {
	mu           sync.Mutex
	closed       bool
	execCalls    []string
	autocommit   bool
	autoflush    bool
}

func newMockSession() *MockSession {
	return &MockSession{autocommit: false, autoflush: false}
}

func (m *MockSession) Exec(query string, _ ...interface{}) (sql.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.execCalls = append(m.execCalls, query)
	return nil, nil
}

func (m *MockSession) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *MockSession) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// ---------------------------------------------------------------------------
// Fake schema registry (mirrors SQLAlchemy metadata create_all / drop_all)
// ---------------------------------------------------------------------------

type SchemaManager interface {
	CreateAll() error
	DropAll() error
}

type MockSchemaManager struct {
	mu          sync.Mutex
	createCalls int
	dropCalls   int
}

func (s *MockSchemaManager) CreateAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.createCalls++
	return nil
}

func (s *MockSchemaManager) DropAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropCalls++
	return nil
}

// ---------------------------------------------------------------------------
// Fake dependency-override registry (mirrors FastAPI app.dependency_overrides)
// ---------------------------------------------------------------------------

type DependencyRegistry interface {
	Override(key string, provider func() DBSession)
	Clear()
	Get(key string) (func() DBSession, bool)
}

type MockDependencyRegistry struct {
	mu        sync.Mutex
	overrides map[string]func() DBSession
}

func newMockDependencyRegistry() *MockDependencyRegistry {
	return &MockDependencyRegistry{overrides: make(map[string]func() DBSession)}
}

func (r *MockDependencyRegistry) Override(key string, provider func() DBSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.overrides[key] = provider
}

func (r *MockDependencyRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.overrides = make(map[string]func() DBSession)
}

func (r *MockDependencyRegistry) Get(key string) (func() DBSession, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.overrides[key]
	return p, ok
}

// ---------------------------------------------------------------------------
// dbSessionFixture helper
// (Go equivalent of the Python db_session_fixture pytest fixture)
// ---------------------------------------------------------------------------

// dbSessionFixture sets up an isolated session for a single test.
// It calls schema.CreateAll(), returns the session, and registers a
// t.Cleanup that closes the session and calls schema.DropAll().
func dbSessionFixture(t *testing.T, schema SchemaManager, session DBSession) DBSession {
	t.Helper()

	require.NoError(t, schema.CreateAll(), "CreateAll must succeed before yielding session")

	t.Cleanup(func() {
		err := session.Close()
		assert.NoError(t, err, "session.Close should not error during teardown")
		err = schema.DropAll()
		assert.NoError(t, err, "DropAll should not error during teardown")
	})

	return session
}

// ---------------------------------------------------------------------------
// clientFixture helper
// (Go equivalent of the Python client_fixture pytest fixture)
// ---------------------------------------------------------------------------

// clientFixture overrides the DB dependency in the registry, wires a
// test HTTP server, and registers teardown that clears overrides.
func clientFixture(
	t *testing.T,
	session DBSession,
	registry DependencyRegistry,
	handler http.Handler,
) *httptest.Server {
	t.Helper()

	const dbKey = "get_db"
	registry.Override(dbKey, func() DBSession { return session })

	srv := httptest.NewServer(handler)
	t.Cleanup(func() {
		srv.Close()
		registry.Clear()
	})

	return srv
}

// ---------------------------------------------------------------------------
// Table-driven tests
// ---------------------------------------------------------------------------

// ── db_session_fixture ─────────────────────────────────────────────────────

func TestDBSessionFixture_TablesCreatedBeforeYield(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "creates_all_tables_before_returning_session"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schema := &MockSchemaManager{}
			session := newMockSession()

			_ = dbSessionFixture(t, schema, session)

			assert.Equal(t, 1, schema.createCalls,
				"CreateAll must be called exactly once before the session is returned")
			assert.Equal(t, 0, schema.dropCalls,
				"DropAll must not be called before the test completes")
		})
	}
}

func TestDBSessionFixture_SessionClosedAndTablesDroppedAfterTest(t *testing.T) {
	tests := []struct {
		name                 string
		simulateTestPanic    bool
	}{
		{name: "successful_test_triggers_teardown", simulateTestPanic: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schema := &MockSchemaManager{}
			session := newMockSession()

			// Run the fixture inside a sub-test so we can inspect teardown
			// after the inner test completes.
			var innerSchema *MockSchemaManager
			var innerSession *MockSession

			t.Run("inner", func(inner *testing.T) {
				innerSchema = schema
				innerSession = session
				_ = dbSessionFixture(inner, innerSchema, innerSession)
				_ = tc.simulateTestPanic // suppress "unused" warning
			})

			// After the inner test finishes, Cleanup functions have run.
			assert.True(t, innerSession.isClosed(),
				"session must be closed during teardown")
			assert.Equal(t, 1, innerSchema.dropCalls,
				"DropAll must be called exactly once during teardown")
		})
	}
}

func TestDBSessionFixture_SessionAutocommitDisabled(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "session_has_autocommit_false"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schema := &MockSchemaManager{}
			session := newMockSession()

			returned := dbSessionFixture(t, schema, session)

			// Assert the returned session is the same object (same settings)
			ms, ok := returned.(*MockSession)
			require.True(t, ok)
			assert.False(t, ms.autocommit, "session autocommit must be disabled")
		})
	}
}

func TestDBSessionFixture_SessionAutoflushDisabled(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "session_has_autoflush_false"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schema := &MockSchemaManager{}
			session := newMockSession()

			returned := dbSessionFixture(t, schema, session)

			ms, ok := returned.(*MockSession)
			require.True(t, ok)
			assert.False(t, ms.autoflush, "session autoflush must be disabled")
		})
	}
}

func TestDBSessionFixture_YieldedSessionIsBound(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "returned_session_is_the_configured_session"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schema := &MockSchemaManager{}
			session := newMockSession()

			returned := dbSessionFixture(t, schema, session)

			assert.Equal(t, session, returned,
				"the session returned must be exactly the one passed in")
		})
	}
}

func TestDBSessionFixture_IsolationBetweenTests(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "first_test"},
		{name: "second_test"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schema := &MockSchemaManager{}
			session := newMockSession()

			t.Run("inner", func(inner *testing.T) {
				_ = dbSessionFixture(inner, schema, session)
			})

			// Each run gets a fresh schema (createCalls resets per iteration)
			assert.Equal(t, 1, schema.createCalls,
				tc.name+": each test must create its own schema")
			assert.Equal(t, 1, schema.dropCalls,
				tc.name+": each test must drop its own schema")
		})
	}
}

// ── client_fixture ─────────────────────────────────────────────────────────

func TestClientFixture_OverrideSetBeforeClientYielded(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "dependency_overridden_before_client_is_returned"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := newMockSession()
			registry := newMockDependencyRegistry()
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			srv := clientFixture(t, session, registry, handler)
			require.NotNil(t, srv)

			provider, ok := registry.Get("get_db")
			assert.True(t, ok, "get_db must be overridden when client is active")
			require.NotNil(t, provider)

			// The provider must return the exact test session
			got := provider()
			assert.Equal(t, session, got,
				"the overridden provider must return the test session")
		})
	}
}

func TestClientFixture_RequestsUseTestSession(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{name: "GET /health uses test session", path: "/health", expectedStatus: http.StatusOK},
		{name: "GET /data uses test session", path: "/data", expectedStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := newMockSession()
			registry := newMockDependencyRegistry()

			// Handler simulates an endpoint that consults the registry for a session.
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				provider, ok := registry.Get("get_db")
				if !ok {
					http.Error(w, "no session", http.StatusInternalServerError)
					return
				}
				sess := provider()
				if sess == nil {
					http.Error(w, "nil session", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
			})

			srv := clientFixture(t, session, registry, handler)

			resp, err := http.Get(srv.URL + tc.path)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
		})
	}
}

func TestClientFixture_TeardownClearsOverrides(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "overrides_cleared_after_test_completes"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := newMockSession()
			registry := newMockDependencyRegistry()
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			var sharedRegistry *MockDependencyRegistry

			t.Run("inner", func(inner *testing.T) {
				sharedRegistry = registry
				srv := clientFixture(inner, session, sharedRegistry, handler)
				require.NotNil(t, srv)
			})

			// After inner test teardown the registry must be empty
			_, ok := sharedRegistry.Get("get_db")
			assert.False(t, ok,
				"dependency overrides must be cleared during teardown")
		})
	}
}

func TestClientFixture_ServerClosedAfterTest(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "test_server_is_closed_after_test_completes"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := newMockSession()
			registry := newMockDependencyRegistry()
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			var serverURL string

			t.Run("inner", func(inner *testing.T) {
				srv := clientFixture(inner, session, registry, handler)
				serverURL = srv.URL
			})

			// After inner test the server must be closed; new requests fail.
			_, err := http.Get(serverURL + "/ping")
			assert.Error(t, err,
				"requests to a closed test server must return an error")
		})
	}
}

func TestClientFixture_SharedSessionBetweenClientAndFixture(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "client_and_session_fixture_share_same_instance"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := newMockSession()
			registry := newMockDependencyRegistry()
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			_ = clientFixture(t, session, registry, handler)

			provider, ok := registry.Get("get_db")
			require.True(t, ok)

			// Must be the exact same pointer, not a copy
			resolved := provider()
			assert.Same(t, session, resolved,
				"the session injected into the registry must be the same object as the fixture session")
		})
	}
}

// ── global invariants ──────────────────────────────────────────────────────

func Test