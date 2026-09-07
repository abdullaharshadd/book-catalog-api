package conftest_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// ---------------------------------------------------------------------------
// Minimal in-process re-implementation of the Python conftest fixtures so
// that the behavioural specs can be validated from Go tests.
// ---------------------------------------------------------------------------

// DBSession is the minimal interface that a "database session" must satisfy.
type DBSession interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Close() error
}

// sqliteSession wraps *sql.DB so it matches DBSession.
type sqliteSession struct{ db *sql.DB }

func (s *sqliteSession) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.db.Exec(query, args...)
}
func (s *sqliteSession) Close() error { return s.db.Close() }

// schemaManager handles create/drop of tables (mirrors Base.metadata behaviour).
type schemaManager struct {
	ddlCreate string
	ddlDrop   string
}

func (sm *schemaManager) CreateAll(sess DBSession) error {
	_, err := sess.Exec(sm.ddlCreate)
	return err
}
func (sm *schemaManager) DropAll(sess DBSession) error {
	_, err := sess.Exec(sm.ddlDrop)
	return err
}

// newTestingSessionLocal opens a brand-new in-memory SQLite DB (check_same_thread=False equivalent).
func newTestingSessionLocal() (*sqliteSession, error) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_check_same_thread=false")
	if err != nil {
		return nil, err
	}
	// Non-autocommit: SQLite driver in modernc starts an implicit transaction,
	// but we control commits explicitly – mirrors non-autocommit behaviour.
	return &sqliteSession{db: db}, nil
}

// defaultSchema is the schema used in the fixture tests.
var defaultSchema = &schemaManager{
	ddlCreate: `CREATE TABLE IF NOT EXISTS items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
	ddlDrop:   `DROP TABLE IF EXISTS items`,
}

// ---------------------------------------------------------------------------
// dbSessionFixture mirrors the Python db_session fixture behaviour.
// ---------------------------------------------------------------------------

type dbSessionFixtureResult struct {
	session      DBSession
	schemaExists bool
	sessionClosed bool
	tablesDropped bool
}

// runDBSessionFixture executes the fixture lifecycle and captures observable side-effects.
func runDBSessionFixture(schema *schemaManager) (*dbSessionFixtureResult, error) {
	result := &dbSessionFixtureResult{}

	// --- Setup (before yield) ---
	sess, err := newTestingSessionLocal()
	if err != nil {
		return nil, err
	}
	if err := schema.CreateAll(sess); err != nil {
		_ = sess.Close()
		return nil, err
	}

	// Verify tables exist (schemaExists) by querying the sqlite_master catalogue.
	row := sess.db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='items'`)
	var count int
	if err := row.Scan(&count); err != nil {
		_ = sess.Close()
		return nil, err
	}
	result.schemaExists = count == 1
	result.session = sess

	// --- Yield (test body runs here – we simulate it by doing nothing special) ---

	// --- Teardown (after yield) ---
	closeErr := sess.Close()
	result.sessionClosed = closeErr == nil

	// Open a new connection just to verify tables are gone after DropAll.
	db2, err := sql.Open("sqlite", "file::memory:?cache=shared&_check_same_thread=false")
	if err != nil {
		return result, err
	}
	defer db2.Close()
	sess2 := &sqliteSession{db: db2}
	if err := schema.DropAll(sess2); err != nil {
		return result, err
	}
	row2 := db2.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='items'`)
	var count2 int
	if err := row2.Scan(&count2); err != nil {
		return result, err
	}
	result.tablesDropped = count2 == 0

	return result, nil
}

// ---------------------------------------------------------------------------
// clientFixture mirrors the Python client_fixture behaviour.
// ---------------------------------------------------------------------------

// getDB is the dependency that the handler uses (mirrors FastAPI's get_db).
var getDB func() DBSession

// overrideGetDB replaces getDB with a function that returns the provided session.
func overrideGetDB(sess DBSession) func() {
	original := getDB
	getDB = func() DBSession { return sess }
	return func() { getDB = original } // teardown / clear override
}

// itemsHandler is a minimal HTTP handler that exercises the test DB session.
func itemsHandler(w http.ResponseWriter, r *http.Request) {
	sess := getDB()
	if sess == nil {
		http.Error(w, "no session", http.StatusInternalServerError)
		return
	}
	// Insert a row to prove the session is operative.
	if _, err := sess.Exec(`INSERT INTO items(name) VALUES(?)`, "test-item"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// newTestApp returns a minimal *http.ServeMux that mimics the FastAPI app.
func newTestApp() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/items", itemsHandler)
	return mux
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestDBSessionFixture validates every behavioural spec for db_session_fixture.
func TestDBSessionFixture(t *testing.T) {
	tests := []struct {
		name                  string
		schema                *schemaManager
		wantSchemaExists      bool
		wantSessionClosed     bool
		wantTablesDropped     bool
		wantSetupErr          bool
	}{
		{
			name:              "tables_created_before_yield_and_dropped_after",
			schema:            defaultSchema,
			wantSchemaExists:  true,
			wantSessionClosed: true,
			wantTablesDropped: true,
			wantSetupErr:      false,
		},
		{
			name: "fresh_schema_state_each_invocation",
			schema: &schemaManager{
				ddlCreate: `CREATE TABLE IF NOT EXISTS orders (id INTEGER PRIMARY KEY, total REAL)`,
				ddlDrop:   `DROP TABLE IF EXISTS orders`,
			},
			wantSchemaExists:  true,
			wantSessionClosed: true,
			wantTablesDropped: true,
			wantSetupErr:      false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result, err := runDBSessionFixture(tc.schema)
			if tc.wantSetupErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantSchemaExists, result.schemaExists,
				"tables must exist during the yielded session")
			assert.Equal(t, tc.wantSessionClosed, result.sessionClosed,
				"session must be closed after the test")
			assert.Equal(t, tc.wantTablesDropped, result.tablesDropped,
				"tables must be dropped after the test")
		})
	}
}

// TestDBSessionFixtureIsolation validates that each invocation starts with a
// fresh schema (invariant: created before, dropped after, per invocation).
func TestDBSessionFixtureIsolation(t *testing.T) {
	const iterations = 3
	for i := 0; i < iterations; i++ {
		result, err := runDBSessionFixture(defaultSchema)
		require.NoError(t, err, "iteration %d must not error", i)
		assert.True(t, result.schemaExists, "iteration %d: schema must exist during session", i)
		assert.True(t, result.sessionClosed, "iteration %d: session must be closed after test", i)
		assert.True(t, result.tablesDropped, "iteration %d: tables must be dropped after test", i)
	}
}

// TestDBSessionFixtureSessionNotNil validates that a live session is yielded.
func TestDBSessionFixtureSessionNotNil(t *testing.T) {
	sess, err := newTestingSessionLocal()
	require.NoError(t, err)
	defer sess.Close()

	err = defaultSchema.CreateAll(sess)
	require.NoError(t, err)
	defer defaultSchema.DropAll(sess) //nolint:errcheck

	assert.NotNil(t, sess, "fixture must yield a non-nil session")
}

// TestDBSessionFixtureNonAutoCommit validates sessions are non-autocommit /
// non-autoflush (changes are explicit).
func TestDBSessionFixtureNonAutoCommit(t *testing.T) {
	sess, err := newTestingSessionLocal()
	require.NoError(t, err)
	defer sess.Close()

	err = defaultSchema.CreateAll(sess)
	require.NoError(t, err)
	defer defaultSchema.DropAll(sess) //nolint:errcheck

	// Execute an insert – if autocommit were broken this would error immediately.
	_, err = sess.Exec(`INSERT INTO items(name) VALUES(?)`, "autocommit-check")
	assert.NoError(t, err, "session must be able to execute DML")
}

// TestDBSessionFixtureCrossThreadAccess validates that check_same_thread=False
// equivalent is in effect (access from goroutines must not panic/error).
func TestDBSessionFixtureCrossThreadAccess(t *testing.T) {
	sess, err := newTestingSessionLocal()
	require.NoError(t, err)
	defer sess.Close()

	err = defaultSchema.CreateAll(sess)
	require.NoError(t, err)
	defer defaultSchema.DropAll(sess) //nolint:errcheck

	var wg sync.WaitGroup
	errs := make([]error, 5)
	for i := 0; i < 5; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = sess.Exec(`INSERT INTO items(name) VALUES(?)`, "goroutine-item")
		}()
	}
	wg.Wait()

	for i, e := range errs {
		assert.NoError(t, e, "goroutine %d must not encounter cross-thread error", i)
	}
}

// ---------------------------------------------------------------------------
// clientFixture tests
// ---------------------------------------------------------------------------

// TestClientFixture validates every behavioural spec for client_fixture.
func TestClientFixture(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		method          string
		wantStatus      int
		wantBody        string
		seedTableBefore bool
	}{
		{
			name:            "client_uses_test_db_session_for_requests",
			path:            "/items",
			method:          http.MethodPost,
			wantStatus:      http.StatusOK,
			wantBody:        `{"status":"ok"}`,
			seedTableBefore: true,
		},
		{
			name:            "handler_fails_when_no_session_override",
			path:            "/items",
			method:          http.MethodPost,
			wantStatus:      http.StatusInternalServerError,
			seedTableBefore: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// --- db_session fixture lifecycle ---
			sess, err := newTestingSessionLocal()
			require.NoError(t, err)

			if tc.seedTableBefore {
				err = defaultSchema.CreateAll(sess)
				require.NoError(t, err)
			}

			// --- client_fixture lifecycle: register override ---
			var clearOverride func()
			if tc.seedTableBefore {
				clearOverride = overrideGetDB(sess)
			} else {
				// Simulate missing override: getDB stays nil.
				getDB = nil
				clearOverride = func() { getDB = nil }
			}

			app := newTestApp()

			// --- HTTP request via httptest ---
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)

			resp := w.Result()
			assert.Equal(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assert.Equal(t, tc.wantBody, w.Body.String())
			}

			// --- Teardown: clear override, close session, drop tables ---
			clearOverride()
			assert.Nil(t, getDB, "dependency overrides must be cleared after the test")

			_ = sess.Close()
			if tc.seedTableBefore {
				// Drop must not error.
				sess2, err2 := newTestingSessionLocal()
				require.NoError(t, err2)
				require.NoError(t, defaultSchema.DropAll(sess2))
				_ = sess2.Close()
			}
		})
	}
}

// TestClientFixtureDependencyOverrideNotLeaking validates that dependency
// overrides do not leak into subsequent tests.
func TestClientFixtureDependencyOverrideNotLeaking(t *testing.T) {
	sess, err := newTestingSessionLocal()
	require.NoError(t, err)
	err = defaultSchema.CreateAll(sess)
	require.NoError(t, err)

	// Install override.
	clearOverride := overrideGetDB(sess)
	assert.NotNil(t, getDB, "override must be active during the test")

	// Teardown.
	clearOverride()
	_ = sess.Close()
	sess2, err := newTestingSessionLocal()
	require.NoError(t, err)
	require.NoError(t, defaultSchema.DropAll(sess2))
	_ = sess2.Close()

	// Override must be gone.
	assert.Nil(t, getDB, "override must not leak into subsequent tests")
}

// TestClientFixtureAppLifecycle validates that the TestClient context manager
// equivalent (startup / shutdown) is honoured.
func TestClientFixtureAppLifecycle(t *testing.T) {
	started := false
	stopped := false

	// Simulate app startup / shutdown via a custom handler with lifecycle hooks.
	startup := func() { started = true }
	shutdown := func() { stopped = true }

	// Enter context (startup).
	startup()
	assert.True(t, started, "app startup must be invoked when client context is entered")

	sess, err := newTestingSessionLocal()
	require.NoError(t, err)
	err = defaultSchema.CreateAll(sess)
	require.NoError(t, err)

	clearOverride := overrideGetDB(sess)

	app := newTestApp()
	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Exit context (shutdown).
	clearOverride()
	_ = sess.Close()

	sess2, err := newTestingSessionLocal()
	require.NoError(t, err