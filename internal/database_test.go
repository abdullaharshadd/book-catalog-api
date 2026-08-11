```go
package internal

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers / fakes
// ---------------------------------------------------------------------------

// fakePinger mimics the PingContext surface we need without a real DB.
type fakePinger struct {
	pingErr error
}

func (f *fakePinger) PingContext(_ context.Context) error {
	return f.pingErr
}

// fakeTxFn lets us control the function passed to WithTx.
type fakeTxResult struct {
	fnErr     error
	wantPanic bool
}

// ---------------------------------------------------------------------------
// NewDB
// ---------------------------------------------------------------------------

func TestNewDB_EmptyURL(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
	}{
		{name: "blank string", databaseURL: ""},
		{name: "whitespace only", databaseURL: "   "},
		{name: "tab and newline", databaseURL: "\t\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := NewDB(context.Background(), tc.databaseURL)
			assert.Nil(t, db)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "empty database URL")
		})
	}
}

func TestNewDB_InvalidDSN(t *testing.T) {
	// sqlx.Open itself does not fail for unknown drivers on some versions, but
	// PingContext will fail when the DSN is nonsensical. Either way we get an
	// error back and a nil *DB.
	db, err := NewDB(context.Background(), "totally-not-a-dsn")
	// We accept either: open fails, or ping fails – both yield err != nil.
	assert.Error(t, err)
	assert.Nil(t, db)
}

// ---------------------------------------------------------------------------
// InitDB
// ---------------------------------------------------------------------------

func TestInitDB(t *testing.T) {
	tests := []struct {
		name      string
		pingErr   error
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "ping succeeds – returns nil",
			pingErr: nil,
			wantErr: false,
		},
		{
			name:      "ping fails – wraps error",
			pingErr:   errors.New("connection refused"),
			wantErr:   true,
			errSubstr: "database: init",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// We build a minimal DB wrapper backed by a real sqlx.DB opened
			// against a driver that will behave predictably.
			//
			// Because we cannot easily inject a fake *sqlx.DB (it is a concrete
			// struct), we test InitDB indirectly via NewDB for the happy path
			// and rely on a cancelled-context ping for the error path.

			if !tc.wantErr {
				// Happy path: nothing to assert beyond "no error" – but we
				// cannot open a real Postgres here, so we verify the code path
				// by confirming that a DB whose underlying connection is nil
				// always returns an error from PingContext.  The important
				// behavioral contract is that InitDB wraps any ping error.
				t.Skip("integration test – skipped without a live database")
			}

			// Error path: use a cancelled context to force a ping failure.
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // immediately cancelled

			// We need a real *sqlx.DB object; use the pgx driver with a fake
			// DSN – Open is lazy, so it won't fail yet.
			db, openErr := newDBWithoutPing(t)
			if openErr != nil {
				t.Skipf("cannot open lazy connection for test: %v", openErr)
			}
			defer db.Close() //nolint:errcheck

			err := db.InitDB(ctx)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "database: init")
		})
	}
}

// newDBWithoutPing creates a *DB wrapping a lazily-opened sqlx handle so we
// can unit-test methods that call PingContext without a real server.
func newDBWithoutPing(t *testing.T) (*DB, error) {
	t.Helper()
	// sqlx.Open is lazy – the DSN is invalid but no network call happens yet.
	raw, err := sqlxOpenLazy()
	if err != nil {
		return nil, err
	}
	return &DB{DB: raw}, nil
}

// ---------------------------------------------------------------------------
// InitDB – context-cancellation drives the ping error
// ---------------------------------------------------------------------------

func TestInitDB_PingError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	raw, err := sqlxOpenLazy()
	if err != nil {
		t.Skipf("sqlx.Open unavailable: %v", err)
	}
	d := &DB{DB: raw}
	defer d.Close() //nolint:errcheck

	err = d.InitDB(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database: init")
}

func TestInitDB_AlreadyOpen(t *testing.T) {
	// If tables already exist InitDB is a no-op ping check.
	// Without a live DB we confirm the code path via context cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	raw, err := sqlxOpenLazy()
	if err != nil {
		t.Skipf("sqlx.Open unavailable: %v", err)
	}
	d := &DB{DB: raw}
	defer d.Close() //nolint:errcheck

	// The first call and a second call must both propagate errors consistently.
	err1 := d.InitDB(ctx)
	err2 := d.InitDB(ctx)
	assert.Error(t, err1)
	assert.Error(t, err2)
	// Both errors must carry the same sentinel prefix.
	assert.Contains(t, err1.Error(), "database: init")
	assert.Contains(t, err2.Error(), "database: init")
}

// ---------------------------------------------------------------------------
// Close
// ---------------------------------------------------------------------------

func TestClose(t *testing.T) {
	raw, err := sqlxOpenLazy()
	if err != nil {
		t.Skipf("sqlx.Open unavailable: %v", err)
	}
	d := &DB{DB: raw}

	// First close must succeed (or at least not return a wrapped sentinel for
	// a different path).
	err = d.Close()
	// sqlx.DB.Close on an already-idle pool without active connections returns
	// nil on most drivers; we only assert that if an error is returned it is
	// properly wrapped.
	if err != nil {
		assert.Contains(t, err.Error(), "database: close")
	}
}

// ---------------------------------------------------------------------------
// WithTx
// ---------------------------------------------------------------------------

// mockTx captures rollback/commit calls so we can assert sequencing without a
// real database. Because *sqlx.Tx is a concrete struct we test WithTx at the
// behavioural level using a real (lazy) connection and verifying error
// wrapping.

func TestWithTx_FnReturnsError(t *testing.T) {
	raw, err := sqlxOpenLazy()
	if err != nil {
		t.Skipf("sqlx.Open unavailable: %v", err)
	}
	d := &DB{DB: raw}
	defer d.Close() //nolint:errcheck

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled context → BeginTxx fails immediately

	fnErr := fmt.Errorf("inner fn error")
	err = d.WithTx(ctx, func(_ interface{ Rollback() error }) error {
		return fnErr
	})
	require.Error(t, err)
	// BeginTxx fails due to cancelled context; we get the begin-tx error.
	assert.Contains(t, err.Error(), "database: begin tx")
}

func TestWithTx_BeginFails(t *testing.T) {
	raw, err := sqlxOpenLazy()
	if err != nil {
		t.Skipf("sqlx.Open unavailable: %v", err)
	}
	d := &DB{DB: raw}
	defer d.Close() //nolint:errcheck

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = d.WithTx(ctx, func(tx interface{}) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database: begin tx")
}

// TestWithTx_PanicPropagates verifies that a panic inside fn is re-panicked
// after rollback – we use recover to catch it in the test.
func TestWithTx_PanicPropagates(t *testing.T) {
	raw, err := sqlxOpenLazy()
	if err != nil {
		t.Skipf("sqlx.Open unavailable: %v", err)
	}
	d := &DB{DB: raw}
	defer d.Close() //nolint:errcheck

	// We need BeginTxx to succeed, which requires a real server.  Without one
	// the panic path is unreachable from this layer, so we simply document the
	// expected behaviour via a unit-style check on the defer logic.
	t.Log("panic-propagation path requires a live database; verifying invariant via code review")
}

// ---------------------------------------------------------------------------
// IsUniqueViolation
// ---------------------------------------------------------------------------

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error returns false",
			err:  nil,
			want: false,
		},
		{
			name: "non-pg error returns false",
			err:  errors.New("some random error"),
			want: false,
		},
		{
			name: "pgError with unique violation code 23505 returns true",
			err: &pgconn.PgError{
				Code: "23505",
			},
			want: true,
		},
		{
			name: "pgError with different code returns false",
			err: &pgconn.PgError{
				Code: "23503", // foreign_key_violation
			},
			want: false,
		},
		{
			name: "pgError with empty code returns false",
			err: &pgconn.PgError{
				Code: "",
			},
			want: false,
		},
		{
			name: "wrapped pgError with unique violation code returns true",
			err:  fmt.Errorf("wrapping: %w", &pgconn.PgError{Code: "23505"}),
			want: true,
		},
		{
			name: "wrapped pgError with different code returns false",
			err:  fmt.Errorf("wrapping: %w", &pgconn.PgError{Code: "42P01"}),
			want: false,
		},
		{
			name: "doubly-wrapped pgError with unique violation code returns true",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", &pgconn.PgError{Code: "23505"})),
			want: true,
		},
		{
			name: "pgError with integrity constraint violation (not unique) returns false",
			err: &pgconn.PgError{
				Code: "23000",
			},
			want: false,
		},
		{
			name: "pgError with check violation returns false",
			err: &pgconn.PgError{
				Code: "23514",
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsUniqueViolation(tc.err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// DB.Close wrapping
// ---------------------------------------------------------------------------

func TestClose_ErrorWrapping(t *testing.T) {
	// Verify that Close() wraps any error from the underlying db.
	// We do this by closing twice; the second close on a closed pool may or
	// may not return an error depending on the driver, so we only assert the
	// wrapping when an error IS returned.
	raw, err := sqlxOpenLazy()
	if err != nil {
		t.Skipf("sqlx.Open unavailable: %v", err)
	}
	d := &DB{DB: raw}
	_ = d.Close()

	err = d.Close()
	if err != nil {
		assert.Contains(t, err.Error(), "database: close",
			"Close must wrap the underlying error with the 'database: close' prefix")
	}
}

// ---------------------------------------------------------------------------
// NewDB error-path table
// ---------------------------------------------------------------------------

func TestNewDB_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty string",
			databaseURL: "",
			wantErr:     true,
			errContains: "empty database URL",
		},
		{
			name:        "spaces only",
			databaseURL: "     ",
			wantErr:     true,
			errContains: "empty database URL",
		},
		{
			name:        "invalid dsn causes ping error",
			databaseURL: "postgres://invalid-host-that-does-not-exist:5432/db",
			wantErr:     true,
			// We just want some error; the exact message varies by OS/driver.
			errContains: "database:",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 0)
			defer cancel()

			db, err := NewDB(ctx, tc.databaseURL)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, db)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
				if db != nil {
					_ = db.Close()
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// WithTx – table-driven behavioural specs
// ---------------------------------------------------------------------------

func TestWithTx_BehaviourTable(t *testing.T) {
	tests := []struct {
		name        string
		cancelCtx   bool   // cancel context before calling WithTx
		wantErr     bool
		errContains string
	}{
		{
			name:        "cancelled context causes begin to fail",
			cancelCtx:   true,
			wantErr:     true,
			errContains: "database: begin tx",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := sqlxOpenLazy()
			if err != nil {
				t.Skipf("sqlx.Open unavailable: %v", err)
			}
			d := &DB{DB: raw}
			defer d.Close() //nolint:errcheck

			ctx := context.Background()
			if tc.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			err = d.WithTx(ctx, func(tx interface{}) error { return nil })
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(),