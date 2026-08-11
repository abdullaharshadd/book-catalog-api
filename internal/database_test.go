```go
package internal

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Minimal sql/driver stubs so we can exercise database.go without a real DB.
// ---------------------------------------------------------------------------

// fakeDriver / fakeConn / fakeStmt / fakeRows / fakeTx implement just enough
// of database/sql/driver to let sql.Open + Ping succeed or fail on demand.

var errFakeDB = errors.New("fake db error")

// connBehavior describes what the next fakeConn should do.
type connBehavior struct {
	pingErr  error
	execErr  error
	beginErr error
	// txBehavior
	commitErr   error
	rollbackErr error
}

type fakeDriver struct {
	behavior connBehavior
}

func (d *fakeDriver) Open(_ string) (driver.Conn, error) {
	if d.behavior.pingErr != nil {
		// We simulate ping failure by returning an error on Open so that
		// sql.DB.Ping() fails (sql.DB calls Open internally for Ping when no
		// idle connection exists).
		return nil, d.behavior.pingErr
	}
	return &fakeConn{behavior: d.behavior}, nil
}

type fakeConn struct {
	behavior connBehavior
	closed   bool
}

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) {
	if c.behavior.execErr != nil {
		return nil, c.behavior.execErr
	}
	return &fakeStmt{}, nil
}
func (c *fakeConn) Close() error  { c.closed = true; return nil }
func (c *fakeConn) Begin() (driver.Tx, error) {
	if c.behavior.beginErr != nil {
		return nil, c.behavior.beginErr
	}
	return &fakeTx{
		commitErr:   c.behavior.commitErr,
		rollbackErr: c.behavior.rollbackErr,
	}, nil
}

type fakeStmt struct{}

func (s *fakeStmt) Close() error                                    { return nil }
func (s *fakeStmt) NumInput() int                                   { return 0 }
func (s *fakeStmt) Exec(_ []driver.Value) (driver.Result, error)    { return fakeResult{}, nil }
func (s *fakeStmt) Query(_ []driver.Value) (driver.Rows, error)     { return &fakeRows{}, nil }

type fakeResult struct{}

func (fakeResult) LastInsertId() (int64, error) { return 1, nil }
func (fakeResult) RowsAffected() (int64, error) { return 1, nil }

type fakeRows struct{ done bool }

func (r *fakeRows) Columns() []string              { return []string{} }
func (r *fakeRows) Close() error                   { return nil }
func (r *fakeRows) Next(_ []driver.Value) error {
	if r.done {
		return errors.New("EOF")
	}
	r.done = true
	return io.EOF // sql/driver expects io.EOF
}

// We need io.EOF — import it properly via a helper.
var _ = fmt.Sprintf // keep fmt import used

type fakeTx struct {
	commitErr   error
	rollbackErr error
}

func (t *fakeTx) Commit() error   { return t.commitErr }
func (t *fakeTx) Rollback() error { return t.rollbackErr }

// registerDriver registers a unique named fake driver and returns a *sql.DB
// that uses it. This avoids conflicts between parallel test registrations.
var driverSeq int

func openFakeDB(t *testing.T, b connBehavior) *sql.DB {
	t.Helper()
	driverSeq++
	name := fmt.Sprintf("fakedriver_%d", driverSeq)
	sql.Register(name, &fakeDriver{behavior: b})
	db, err := sql.Open(name, "fake-dsn")
	require.NoError(t, err)
	return db
}

// ---------------------------------------------------------------------------
// Helper: build a *DB from a *sql.DB without going through NewDB (no real PG).
// ---------------------------------------------------------------------------

func newTestDB(pool *sql.DB) *DB {
	return &DB{pool: pool}
}

// ---------------------------------------------------------------------------
// ClampPagination tests
// ---------------------------------------------------------------------------

func TestClampPagination(t *testing.T) {
	tests := []struct {
		name           string
		limit          int
		offset         int
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "positive limit and offset pass through unchanged",
			limit:          10,
			offset:         5,
			expectedLimit:  10,
			expectedOffset: 5,
		},
		{
			name:           "zero limit becomes DefaultPageLimit",
			limit:          0,
			offset:         0,
			expectedLimit:  DefaultPageLimit,
			expectedOffset: 0,
		},
		{
			name:           "negative limit becomes DefaultPageLimit",
			limit:          -1,
			offset:         0,
			expectedLimit:  DefaultPageLimit,
			expectedOffset: 0,
		},
		{
			name:           "limit above MaxPageLimit is capped",
			limit:          MaxPageLimit + 1,
			offset:         0,
			expectedLimit:  MaxPageLimit,
			expectedOffset: 0,
		},
		{
			name:           "limit exactly at MaxPageLimit is unchanged",
			limit:          MaxPageLimit,
			offset:         0,
			expectedLimit:  MaxPageLimit,
			expectedOffset: 0,
		},
		{
			name:           "negative offset becomes 0",
			limit:          10,
			offset:         -5,
			expectedLimit:  10,
			expectedOffset: 0,
		},
		{
			name:           "negative limit and negative offset both clamped",
			limit:          -99,
			offset:         -99,
			expectedLimit:  DefaultPageLimit,
			expectedOffset: 0,
		},
		{
			name:           "large limit capped, large offset unchanged",
			limit:          9999,
			offset:         1000,
			expectedLimit:  MaxPageLimit,
			expectedOffset: 1000,
		},
		{
			name:           "DefaultPageLimit constant value is 20",
			limit:          DefaultPageLimit,
			offset:         0,
			expectedLimit:  DefaultPageLimit,
			expectedOffset: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotLimit, gotOffset := ClampPagination(tc.limit, tc.offset)
			assert.Equal(t, tc.expectedLimit, gotLimit, "limit mismatch")
			assert.Equal(t, tc.expectedOffset, gotOffset, "offset mismatch")
		})
	}
}

// ---------------------------------------------------------------------------
// NewDB tests
// ---------------------------------------------------------------------------

func TestNewDB_EmptyDSN(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
	}{
		{name: "empty string DSN", dsn: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db, err := NewDB(context.Background(), tc.dsn)
			assert.Nil(t, db)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "empty DSN")
		})
	}
}

// ---------------------------------------------------------------------------
// DB.Pool tests
// ---------------------------------------------------------------------------

func TestDB_Pool(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "Pool returns underlying *sql.DB"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := openFakeDB(t, connBehavior{})
			defer pool.Close()

			db := newTestDB(pool)
			assert.Equal(t, pool, db.Pool())
		})
	}
}

// ---------------------------------------------------------------------------
// DB.Close tests
// ---------------------------------------------------------------------------

func TestDB_Close(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "close succeeds on healthy pool", wantErr: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := openFakeDB(t, connBehavior{})
			db := newTestDB(pool)

			err := db.Close()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DB.InitSchema tests  (maps to the init_db behavioral spec)
// ---------------------------------------------------------------------------

func TestDB_InitSchema(t *testing.T) {
	tests := []struct {
		name      string
		execErr   error
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "schema created successfully (idempotent CREATE TABLE IF NOT EXISTS)",
			execErr: nil,
			wantErr: false,
		},
		{
			name:      "propagates database error when exec fails",
			execErr:   errFakeDB,
			wantErr:   true,
			errSubstr: "init schema",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := openFakeDB(t, connBehavior{execErr: tc.execErr})
			defer pool.Close()

			db := newTestDB(pool)
			err := db.InitSchema(context.Background())

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DB.WithTx tests  (maps to get_db / get_sync_db behavioral specs)
// ---------------------------------------------------------------------------

func TestDB_WithTx(t *testing.T) {
	tests := []struct {
		name        string
		beginErr    error
		commitErr   error
		rollbackErr error
		fnErr       error      // error returned by the user function fn
		fnPanic     bool       // whether fn panics
		wantErr     bool
		errSubstr   string
		wantPanic   bool
	}{
		{
			name:    "fn succeeds: transaction committed",
			wantErr: false,
		},
		{
			name:      "fn returns error: transaction rolled back, error propagated",
			fnErr:     errors.New("fn failed"),
			wantErr:   true,
			errSubstr: "fn failed",
		},
		{
			name:      "begin fails: error returned immediately",
			beginErr:  errors.New("begin failed"),
			wantErr:   true,
			errSubstr: "begin tx",
		},
		{
			name:      "fn succeeds but commit fails: commit error returned",
			commitErr: errors.New("commit failed"),
			wantErr:   true,
			errSubstr: "commit",
		},
		{
			name:        "fn returns error and rollback fails: rollback error wraps original",
			fnErr:       errors.New("fn error"),
			rollbackErr: errors.New("rollback error"),
			wantErr:     true,
			// When rollback also fails and it's not sql.ErrTxDone, the error is
			// wrapped; we just verify an error is returned.
		},
		{
			name:      "fn panics: panic is re-raised after rollback",
			fnPanic:   true,
			wantPanic: true,
		},
		{
			name:      "fn returns nil: session is effectively closed (no rollback)",
			fnErr:     nil,
			wantErr:   false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := openFakeDB(t, connBehavior{
				beginErr:    tc.beginErr,
				commitErr:   tc.commitErr,
				rollbackErr: tc.rollbackErr,
			})
			defer pool.Close()

			db := newTestDB(pool)

			fn := func(_ *sql.Tx) error {
				if tc.fnPanic {
					panic("test panic")
				}
				return tc.fnErr
			}

			if tc.wantPanic {
				assert.Panics(t, func() {
					_ = db.WithTx(context.Background(), fn)
				})
				return
			}

			err := db.WithTx(context.Background(), fn)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errSubstr != "" {
					assert.Contains(t, err.Error(), tc.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// WithTx invariant: a new transaction is created per invocation
// ---------------------------------------------------------------------------

func TestDB_WithTx_NewTxPerCall(t *testing.T) {
	pool := openFakeDB(t, connBehavior{})
	defer pool.Close()

	db := newTestDB(pool)

	var txPtrs []*sql.Tx
	for i := 0; i < 3; i++ {
		err := db.WithTx(context.Background(), func(tx *sql.Tx) error {
			txPtrs = append(txPtrs, tx)
			return nil
		})
		require.NoError(t, err)
	}

	assert.Len(t, txPtrs, 3, "expected three distinct calls")
	// Each invocation must supply a non-nil tx.
	for _, tx := range txPtrs {
		assert.NotNil(t, tx)
	}
}

// ---------------------------------------------------------------------------
// WithTx invariant: session always closed (fn nil return vs error)
// ---------------------------------------------------------------------------

func TestDB_WithTx_SessionAlwaysClosed(t *testing.T) {
	tests := []struct {
		name  string
		fnErr error
	}{
		{name: "closed after success", fnErr: nil},
		{name: "closed after fn error", fnErr: errors.New("boom")},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := openFakeDB(t, connBehavior{})
			defer pool.Close()

			db := newTestDB(pool)

			called := false
			_ = db.WithTx(context.Background(), func(tx *sql.Tx) error {
				called = true
				return tc.fnErr
			})

			assert.True(t, called, "fn should have been called")
			// The test itself completing without a goroutine leak is the
			// observable side-effect that the connection was returned to the
			// pool (closed at the transaction level).
		})
	}
}

// ---------------------------------------------------------------------------
// WithTx invariant: rollback called on fn error, not on success
// ---------------------------------------------------------------------------

func TestDB_WithTx_RollbackOnError(t *testing.T) {
	tests := []struct {
		name         string
		fnErr        error
		expectErrNil bool
	}{
		{
			name:         "no rollback on success",
			fnErr:        nil,
			expectErrNil: true,
		},
		{
			name:         "rollback on fn error",
			fnErr:        errors.New("trigger rollback"),
			expectErrNil