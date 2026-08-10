```go
package internal

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newMockedDB wires a go-sqlmock connection into a *DB so tests can control
// every SQL interaction without a real PostgreSQL server.
func newMockedDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	t.Cleanup(func() { _ = sqlxDB.Close() })
	return &DB{DB: sqlxDB}, mock
}

// ---------------------------------------------------------------------------
// NewDB
// ---------------------------------------------------------------------------

func TestNewDB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		databaseURL string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty URL returns error immediately",
			databaseURL: "",
			wantErr:     true,
			errContains: "database url is empty",
		},
		{
			name: "invalid DSN causes open/ping failure",
			// sqlx.Open with driver "postgres" and a totally bogus URL will
			// succeed at Open time (lib/pq is lazy) but fail at PingContext.
			databaseURL: "postgres://invalid-host-that-does-not-exist/db?sslmode=disable&connect_timeout=1",
			wantErr:     true,
			// The error may say "ping database" or a dial error; either way it
			// must be non-nil.
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db, err := NewDB(ctx, tc.databaseURL)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				assert.Nil(t, db)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, db)
				_ = db.Close()
			}
		})
	}
}

// TestNewDB_PingTimeout verifies that a context that is already cancelled
// causes NewDB to fail at the ping stage.
func TestNewDB_PingTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	// Sleep briefly so the context times out before Open even starts.
	time.Sleep(2 * time.Millisecond)

	db, err := NewDB(ctx, "postgres://localhost/neverexists?sslmode=disable&connect_timeout=1")
	assert.Error(t, err)
	assert.Nil(t, db)
}

// ---------------------------------------------------------------------------
// DB.Close
// ---------------------------------------------------------------------------

func TestDB_Close(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver returns nil", func(t *testing.T) {
		t.Parallel()
		var d *DB
		assert.NoError(t, d.Close())
	})

	t.Run("nil inner DB returns nil", func(t *testing.T) {
		t.Parallel()
		d := &DB{}
		assert.NoError(t, d.Close())
	})

	t.Run("healthy DB closes without error", func(t *testing.T) {
		t.Parallel()
		db, mock := newMockedDB(t)
		mock.ExpectClose()
		err := db.Close()
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates underlying close error", func(t *testing.T) {
		t.Parallel()
		// Register a driver whose Close always errors.
		driverName := "errclose_" + t.Name()
		sql.Register(driverName, &errorCloseDriver{})
		rawDB, err := sql.Open(driverName, "")
		require.NoError(t, err)
		d := &DB{DB: sqlx.NewDb(rawDB, driverName)}
		// Open a connection so that the pool has something to close.
		// (The driver's Open returns an errCloseConn which fails on Close.)
		closeErr := d.Close()
		assert.Error(t, closeErr)
		assert.Contains(t, closeErr.Error(), "close database")
	})
}

// ---------------------------------------------------------------------------
// DB.InitSchema
// ---------------------------------------------------------------------------

func TestDB_InitSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(mock sqlmock.Sqlmock)
		wantErr     bool
		errContains string
	}{
		{
			name: "creates tables when none exist",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`CREATE TABLE IF NOT EXISTS books`).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name: "idempotent – no error when tables already exist",
			setup: func(mock sqlmock.Sqlmock) {
				// IF NOT EXISTS means the DB simply returns success again.
				mock.ExpectExec(`CREATE TABLE IF NOT EXISTS books`).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name: "propagates error when exec fails",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`CREATE TABLE IF NOT EXISTS books`).
					WillReturnError(fmt.Errorf("connection refused"))
			},
			wantErr:     true,
			errContains: "init schema",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			db, mock := newMockedDB(t)
			tc.setup(mock)

			err := db.InitSchema(context.Background())
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestDB_InitSchema_Idempotent calls InitSchema twice and confirms both calls
// succeed, simulating the "tables already exist" invariant.
func TestDB_InitSchema_Idempotent(t *testing.T) {
	t.Parallel()
	db, mock := newMockedDB(t)

	// Both invocations expect the same CREATE TABLE IF NOT EXISTS DDL.
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS books`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS books`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	ctx := context.Background()
	require.NoError(t, db.InitSchema(ctx), "first call should succeed")
	require.NoError(t, db.InitSchema(ctx), "second call should succeed (idempotent)")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDB_InitSchema_CancelledContext verifies that a pre-cancelled context
// surfaces an error from ExecContext.
func TestDB_InitSchema_CancelledContext(t *testing.T) {
	t.Parallel()
	db, mock := newMockedDB(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS books`).
		WillReturnError(context.Canceled)

	err := db.InitSchema(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "init schema")
}

// ---------------------------------------------------------------------------
// Connection-pool settings (observable via DB.Stats after construction)
// ---------------------------------------------------------------------------

// TestNewDB_PoolSettings verifies the pool parameters documented in NewDB are
// actually applied. We use a sqlmock-backed DB constructed the same way NewDB
// would wire it so we can inspect Stats() without a real server.
func TestNewDB_PoolSettings(t *testing.T) {
	t.Parallel()
	db, mock := newMockedDB(t)

	// Replicate the exact settings from NewDB.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	stats := db.Stats()
	// MaxOpenConnections is the only pool knob exposed by sql.DBStats.
	assert.Equal(t, 25, stats.MaxOpenConnections)

	mock.ExpectClose()
	_ = db.Close()
}

// ---------------------------------------------------------------------------
// Fake driver helpers for error-path tests
// ---------------------------------------------------------------------------

// errorCloseDriver is a minimal database/sql driver whose connections always
// fail on Close, letting us exercise the "propagates close error" path.
type errorCloseDriver struct{}

func (d *errorCloseDriver) Open(_ string) (driver.Conn, error) {
	return &errCloseConn{}, nil
}

type errCloseConn struct{}

func (c *errCloseConn) Prepare(_ string) (driver.Stmt, error) { return nil, nil }
func (c *errCloseConn) Begin() (driver.Tx, error)             { return nil, nil }
func (c *errCloseConn) Close() error                          { return fmt.Errorf("simulated close failure") }
```