// internal/database_test.go
package database

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/pressly/goose/v3"
	_ "github.com/lib/pq"
	"os"
	"sync"
)

type mockDB struct {
	mock.Mock
}

func (m *mockDB) PingContext(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type mockSQLX struct {
	mock.Mock
}

func (m *mockSQLX) PingContext(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type mockGoose struct {
	mock.Mock
}

func (m *mockGoose) UpDB(db *sql.DB, fs embed.FS, dir string) error {
	args := m.Called(db, fs, dir)
	return args.Error(0)
}

var (
	mockDBInstance *mockDB
	mockAsyncDBInstance *mockSQLX
	mockGooseInstance *mockGoose
)

func TestInitializeDatabase(t *testing.T) {
	type test struct {
		name                 string
		setup                func()
		expectedErr          bool
		expectedSideEffects  []string
	}

	tests := []test{
		{
			name: "success",
			setup: func() {
				mockDBInstance.On("PingContext", mock.Anything).Return(nil)
				mockAsyncDBInstance.On("PingContext", mock.Anything).Return(nil)
				mockGooseInstance.On("UpDB", mock.Anything, mock.Anything, "migrations").Return(nil)
			},
			expectedErr: false,
			expectedSideEffects: []string{"creates tables in the database using the sync engine"},
		},
		{
			name: "db open error",
			setup: func() {
				mockDBInstance.On("Open", mock.Anything, mock.Anything).Return(nil, errors.New("open error"))
			},
			expectedErr: true,
			expectedSideEffects: []string{},
		},
		{
			name: "db ping error",
			setup: func() {
				mockDBInstance.On("PingContext", mock.Anything).Return(errors.New("ping error"))
			},
			expectedErr: true,
			expectedSideEffects: []string{},
		},
		{
			name: "async db connect error",
			setup: func() {
				mockAsyncDBInstance.On("Connect", mock.Anything, mock.Anything).Return(nil, errors.New("connect error"))
			},
			expectedErr: true,
			expectedSideEffects: []string{},
		},
		{
			name: "async db ping error",
			setup: func() {
				mockAsyncDBInstance.On("PingContext", mock.Anything).Return(errors.New("ping error"))
			},
			expectedErr: true,
			expectedSideEffects: []string{},
		},
		{
			name: "migration error",
			setup: func() {
				mockDBInstance.On("PingContext", mock.Anything).Return(nil)
				mockAsyncDBInstance.On("PingContext", mock.Anything).Return(nil)
				mockGooseInstance.On("UpDB", mock.Anything, mock.Anything, "migrations").Return(errors.New("migration error"))
			},
			expectedErr: true,
			expectedSideEffects: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			// Reset global variables for each test
			dbOnce = sync.Once{}
			db = nil
			asyncDB = nil

			err := InitializeDatabase(context.Background())

			if tt.expectedErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			for _, sideEffect := range tt.expectedSideEffects {
				switch sideEffect {
				case "creates tables in the database using the sync engine":
					mockGooseInstance.AssertCalled(t, "UpDB", mock.Anything, mock.Anything, "migrations")
				}
			}
		})
	}
}

func TestGetDB(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dbOnce.Do(func() {
			db = &sql.DB{}
		})

		result := GetDB()
		assert.NotNil(t, result)
	})
}

func TestGetAsyncDB(t *testing.T) {
	type test struct {
		name                 string
		setup                func()
		expectedOutput       *sqlx.DB
		expectedErr          error
		expectedSideEffects  []string
	}

	tests := []test{
		{
			name: "success",
			setup: func() {
				asyncDB = &sqlx.DB{}
			},
			expectedOutput: &sqlx.DB{},
			expectedErr:    nil,
		},
		{
			name: "not initialized",
			setup: func() {},
			expectedOutput: nil,
			expectedErr:    fmt.Errorf("async database not initialized"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			result, err := GetAsyncDB(context.Background())
			assert.Equal(t, tt.expectedOutput, result)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}