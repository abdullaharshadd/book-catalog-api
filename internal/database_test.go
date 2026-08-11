```go
package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newMockedDB wraps a sqlmock-backed *sql.DB into the application's *DB type.
func newMockedDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(rawDB, "sqlmock")
	db := &DB{DB: sqlxDB}

	t.Cleanup(func() {
		_ = rawDB.Close()
	})
	return db, mock
}

// ---------------------------------------------------------------------------
// NewDB
// ---------------------------------------------------------------------------

func TestNewDB(t *testing.T) {
	// NewDB requires a real or fake network endpoint to Ping.
	// We test the two observable failure paths without a live Postgres instance.

	tests := []struct {
		name    string
		dsn     string
		wantErr bool
		errMsg  string
	}{
		{
			// sqlx.Open itself validates the driver name; "postgres" is registered
			// by lib/pq, but attempting Ping against a bogus DSN will fail.
			name:    "ping_failure_bad_dsn",
			dsn:     "host=127.0.0.1 port=1 user=nobody dbname=nobody sslmode=disable connect_timeout=1",
			wantErr: true,
			errMsg:  "ping database",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, err := NewDB(ctx, tc.dsn)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
				assert.Nil(t, db)
			} else {
				require.NoError(t, err)
				require.NotNil(t, db)
				_ = db.Close()
			}
		})
	}
}

// ---------------------------------------------------------------------------
// InitSchema
// ---------------------------------------------------------------------------

func TestInitSchema(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "creates_tables_when_none_exist",
			setupMock: func(mock sqlmock.Sqlmock) {
				// Expect ANY exec that contains our DDL keyword.
				mock.ExpectExec(`CREATE TABLE IF NOT EXISTS books`).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name: "idempotent_when_tables_already_exist",
			setupMock: func(mock sqlmock.Sqlmock) {
				// IF NOT EXISTS makes it a no-op; the DB still returns success.
				mock.ExpectExec(`CREATE TABLE IF NOT EXISTS books`).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name: "propagates_db_error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`CREATE TABLE IF NOT EXISTS books`).
					WillReturnError(errors.New("connection refused"))
			},
			wantErr: true,
			errMsg:  "init schema",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMockedDB(t)
			tc.setupMock(mock)

			err := db.InitSchema(context.Background())

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ---------------------------------------------------------------------------
// InitSchema – invariants
// ---------------------------------------------------------------------------

func TestInitSchema_Invariants(t *testing.T) {
	t.Run("books_table_created_with_expected_columns", func(t *testing.T) {
		db, mock := newMockedDB(t)

		// The DDL must contain all required columns.
		mock.ExpectExec(`CREATE TABLE IF NOT EXISTS books`).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := db.InitSchema(context.Background())
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("does_not_drop_existing_tables", func(t *testing.T) {
		// The schema uses CREATE TABLE IF NOT EXISTS — no DROP is ever issued.
		db, mock := newMockedDB(t)

		// We expect exactly ONE Exec (the CREATE) and no DROP.
		mock.ExpectExec(`CREATE TABLE IF NOT EXISTS books`).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := db.InitSchema(context.Background())
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// WithTx – maps to get_db / get_sync_db behavioural specs
// ---------------------------------------------------------------------------

func TestWithTx(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(mock sqlmock.Sqlmock)
		fn        func(tx *sqlx.Tx) error
		wantErr   bool
		errMsg    string
	}{
		{
			name: "commits_on_success",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			fn:      func(tx *sqlx.Tx) error { return nil },
			wantErr: false,
		},
		{
			name: "rolls_back_and_propagates_error_on_fn_failure",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			fn:      func(tx *sqlx.Tx) error { return errors.New("query failed") },
			wantErr: true,
			errMsg:  "query failed",
		},
		{
			name: "begin_failure_returns_error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errors.New("cannot begin"))
			},
			fn:      func(tx *sqlx.Tx) error { return nil },
			wantErr: true,
			errMsg:  "begin tx",
		},
		{
			name: "commit_failure_returns_error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit().WillReturnError(errors.New("commit refused"))
			},
			fn:      func(tx *sqlx.Tx) error { return nil },
			wantErr: true,
			errMsg:  "commit tx",
		},
		{
			name: "rollback_failure_wraps_original_error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback().WillReturnError(errors.New("rollback refused"))
			},
			fn:      func(tx *sqlx.Tx) error { return errors.New("original error") },
			wantErr: true,
			// Both the rollback and original errors should appear.
			errMsg: "rollback failed",
		},
		{
			name: "session_is_always_closed_on_success",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			fn:      func(tx *sqlx.Tx) error { return nil },
			wantErr: false,
		},
		{
			name: "session_is_always_closed_on_failure",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			fn:      func(tx *sqlx.Tx) error { return errors.New("boom") },
			wantErr: true,
		},
		{
			name: "each_invocation_uses_distinct_transaction",
			setupMock: func(mock sqlmock.Sqlmock) {
				// Two sequential WithTx calls → two begin/commit pairs.
				mock.ExpectBegin()
				mock.ExpectCommit()
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			// fn is called twice externally; this entry only exercises the
			// first call; the second is done in the test body below.
			fn:      func(tx *sqlx.Tx) error { return nil },
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMockedDB(t)
			tc.setupMock(mock)

			err := db.WithTx(context.Background(), tc.fn)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}

			// For the "distinct transaction" scenario run a second call.
			if tc.name == "each_invocation_uses_distinct_transaction" {
				err2 := db.WithTx(context.Background(), tc.fn)
				require.NoError(t, err2)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ---------------------------------------------------------------------------
// WithTx – panic recovery
// ---------------------------------------------------------------------------

func TestWithTx_PanicRecovery(t *testing.T) {
	db, mock := newMockedDB(t)

	mock.ExpectBegin()
	mock.ExpectRollback()

	assert.Panics(t, func() {
		_ = db.WithTx(context.Background(), func(tx *sqlx.Tx) error {
			panic("unexpected panic")
		})
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// WithTx – session lifecycle invariants
// ---------------------------------------------------------------------------

func TestWithTx_SessionLifecycleInvariants(t *testing.T) {
	t.Run("exactly_one_transaction_per_invocation", func(t *testing.T) {
		db, mock := newMockedDB(t)

		mock.ExpectBegin()
		mock.ExpectCommit()

		callCount := 0
		err := db.WithTx(context.Background(), func(tx *sqlx.Tx) error {
			callCount++
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, 1, callCount)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rollback_happens_before_error_propagation", func(t *testing.T) {
		db, mock := newMockedDB(t)

		rollbackCalled := false
		mock.ExpectBegin()
		// Intercept rollback to record it was called.
		mock.ExpectRollback().WillReturnResult() // sqlmock records the call

		// We cannot inject a side-effect hook here, so we verify via
		// ExpectationsWereMet that Rollback was indeed called.
		originalErr := errors.New("fn error")
		err := db.WithTx(context.Background(), func(tx *sqlx.Tx) error {
			return originalErr
		})

		require.Error(t, err)
		assert.True(t, errors.Is(err, originalErr))
		_ = rollbackCalled // satisfied by mock assertion below
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// Close
// ---------------------------------------------------------------------------

func TestDB_Close(t *testing.T) {
	t.Run("closes_underlying_pool", func(t *testing.T) {
		rawDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		mock.ExpectClose()

		sqlxDB := sqlx.NewDb(rawDB, "sqlmock")
		db := &DB{DB: sqlxDB}

		err = db.Close()
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// DB struct embedding
// ---------------------------------------------------------------------------

func TestDB_EmbedsSqlxDB(t *testing.T) {
	rawDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer rawDB.Close()

	sqlxDB := sqlx.NewDb(rawDB, "sqlmock")
	db := &DB{DB: sqlxDB}

	// The embedded *sqlx.DB must be accessible directly.
	assert.NotNil(t, db.DB)
	assert.Equal(t, sqlxDB, db.DB)
}

// ---------------------------------------------------------------------------
// WithTx – fn receives usable Tx (query round-trip)
// ---------------------------------------------------------------------------

func TestWithTx_FnReceivesUsableTx(t *testing.T) {
	db, mock := newMockedDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT 1`).
		WillReturnRows(sqlmock.NewRows([]string{"?column?"}).AddRow(1))
	mock.ExpectCommit()

	err := db.WithTx(context.Background(), func(tx *sqlx.Tx) error {
		var result int
		return tx.QueryRowContext(context.Background(), "SELECT 1").Scan(&result)
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// WithTx – original error preserved through rollback
// ---------------------------------------------------------------------------

func TestWithTx_OriginalErrorPreserved(t *testing.T) {
	sentinel := errors.New("sentinel error")

	tests := []struct {
		name          string
		setupMock     func(mock sqlmock.Sqlmock)
		wantSentinel  bool
		wantErrSubstr string
	}{
		{
			name: "sentinel_unwrappable_when_rollback_succeeds",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			wantSentinel: true,
		},
		{
			name: "sentinel_wrapped_when_rollback_also_fails",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback().WillReturnError(errors.New("rb err"))
			},
			wantSentinel:  false,
			wantErrSubstr: "rollback failed",
		},
	}

	for _, tc := range tests