```go
package internal

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

type MockDB struct {
	*sqlx.DB
}

func (m *MockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return m.DB.ExecContext(ctx, query, args...)
}

func (m *MockDB) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	return m.DB.BeginTxx(ctx, opts)
}

func TestNewDatabase(t *testing.T) {
	tests := []struct {
		name    string
		dbConfig string
		asyncDBConfig string
		mockFunc func(mock sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "Success",
			dbConfig: "postgres://user:password@localhost:5432/dbname?sslmode=disable",
			asyncDBConfig: "postgres://user:password@localhost:5432/dbname?sslmode=disable",
			mockFunc: func(mock sqlmock.Sqlmock) {},
			wantErr: false,
		},
		{
			name: "SyncDBOpenError",
			dbConfig: "invalid_sync_db_url",
			asyncDBConfig: "postgres://user:password@localhost:5432/dbname?sslmode=disable",
			mockFunc: func(mock sqlmock.Sqlmock) {},
			wantErr: true,
		},
		{
			name: "AsyncDBOpenError",
			dbConfig: "postgres://user:password@localhost:5432/dbname?sslmode=disable",
			asyncDBConfig: "invalid_async_db_url",
			mockFunc: func(mock sqlmock.Sqlmock) {},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			viper.Set("DATABASE_URL", test.dbConfig)
			viper.Set("ASYNC_DATABASE_URL", test.asyncDBConfig)

			syncDB, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer syncDB.Close()

			mockDB := &sqlx.DB{DB: syncDB.DB}
			syncDB.(*sqlmock.Sqlmock).EXPECT().PingContext(context.Background()).Return(nil)
			test.mockFunc(syncDB.(*sqlmock.Sqlmock))

			asyncDB, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer asyncDB.Close()

			mockAsyncDB := &sqlx.DB{DB: asyncDB.DB}
			asyncDB.(*sqlmock.Sqlmock).EXPECT().PingContext(context.Background()).Return(nil)
			test.mockFunc(asyncDB.(*sqlmock.Sqlmock))

			db, err := NewDatabase(context.Background())
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
			}
		})
	}
}

func TestInitDB(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(mock sqlmock.Sqlmock)
		wantErr     bool
	}{
		{
			name: "Success",
			mockFunc: func(mock sqlmock.Sqlmock) {
				query := `CREATE TABLE IF NOT EXISTS books .+`
				mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "TableCreationError",
			mockFunc: func(mock sqlmock.Sqlmock) {
				query := `CREATE TABLE IF NOT EXISTS books .+`
				mock.ExpectExec(query).WillReturnError(sql.ErrTxDone)
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			mockDB := &MockDB{DB: db.DB}
			test.mockFunc(db.(*sqlmock.Sqlmock))

			err = mockDB.InitDB(context.Background())
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetDB(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(mock sqlmock.Sqlmock)
		wantErr     bool
	}{
		{
			name: "Success",
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "BeginTxxError",
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(sql.ErrTxDone)
			},
			wantErr: true,
		},
		{
			name: "CommitError",
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit().WillReturnError(sql.ErrTxDone)
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			mockDB := &MockDB{DB: db.DB}
			test.mockFunc(db.(*sqlmock.Sqlmock))

			tx, err := mockDB.GetDB(context.Background())
			if test.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tx)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tx)
				tx.Rollback()
			}
		})
	}
}

func TestGetSyncDB(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(mock sqlmock.Sqlmock)
		wantErr     bool
	}{
		{
			name: "Success",
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "BeginTxxError",
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(sql.ErrTxDone)
			},
			wantErr: true,
		},
		{
			name: "CommitError",
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit().WillReturnError(sql.ErrTxDone)
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			mockDB := &MockDB{DB: db.DB}
			test.mockFunc(db.(*sqlmock.Sqlmock))

			tx, err := mockDB.GetSyncDB()
			if test.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tx)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tx)
				tx.Rollback()
			}
		})
	}
}
```