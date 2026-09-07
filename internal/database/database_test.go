// internal/database_test.go
package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type MockDB struct {
	mock.Mock
}

func (m *MockDB) PingContext(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return &sql.Rows{}, nil
}

type MockSQLXDB struct {
	*MockDB
}

func (m *MockSQLXDB) NewDb(db *sql.DB, driverName string) (*sqlx.DB, error) {
	args := m.Called(db, driverName)
	return args.Get(0).(*sqlx.DB), args.Error(1)
}

func TestInitializeDatabase(t *testing.T) {
	type testCase struct {
		name                  string
		mockPing              func(context.Context) error
		mockCreateSchema      func(context.Context, *sql.DB) error
		expectedErrorMessage  string
		expectedSchemaCreated bool
	}

	testCases := []testCase{
		{
			name:                  "success",
			mockPing:              func(ctx context.Context) error { return nil },
			mockCreateSchema:      func(ctx context.Context, db *sql.DB) error { return nil },
			expectedErrorMessage:  "",
			expectedSchemaCreated: true,
		},
		{
			name:                  "ping fails",
			mockPing:              func(ctx context.Context) error { return errors.New("ping failed") },
			mockCreateSchema:      func(ctx context.Context, db *sql.DB) error { return nil },
			expectedErrorMessage:  "failed to ping sync database: ping failed",
			expectedSchemaCreated: false,
		},
		{
			name:                  "create schema fails",
			mockPing:              func(ctx context.Context) error { return nil },
			mockCreateSchema:      func(ctx context.Context, db *sql.DB) error { return errors.New("schema creation failed") },
			expectedErrorMessage:  "error initializing database: schema creation failed",
			expectedSchemaCreated: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := new(MockDB)
			mockDB.On("PingContext", mock.Anything).Return(tc.mockPing(mock.Anything))
			model.CreateSchema = tc.mockCreateSchema

			err := InitializeDatabase(context.Background())
			if tc.expectedErrorMessage != "" {
				assert.EqualError(t, err, tc.expectedErrorMessage)
			} else {
				assert.NoError(t, err)
			}

			if tc.expectedSchemaCreated {
				mockDB.AssertCalled(t, "PingContext", mock.Anything)
			} else {
				mockDB.AssertNotCalled(t, "PingContext", mock.Anything)
			}
			model.CreateSchema = model.CreateSchemaImpl // reset to original implementation
		})
	}
}

func TestGetSyncDB(t *testing.T) {
	type testCase struct {
		name            string
		expectedErr     error
		expectedSession *sql.DB
	}

	testCases := []testCase{
		{
			name:            "initialized",
			expectedErr:     nil,
			expectedSession: &sql.DB{},
		},
		{
			name:            "not initialized",
			expectedErr:     errors.New("sync database not initialized"),
			expectedSession: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			syncDB = tc.expectedSession
			session, err := GetSyncDB()
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedSession, session)
		})
	}
}

func TestGetAsyncDB(t *testing.T) {
	type testCase struct {
		name            string
		mockPing        func(context.Context) error
		expectedErr     error
		expectedSession *sqlx.DB
	}

	testCases := []testCase{
		{
			name:            "initialized",
			mockPing:        func(ctx context.Context) error { return nil },
			expectedErr:     nil,
			expectedSession: &sqlx.DB{},
		},
		{
			name:            "not initialized",
			mockPing:        func(ctx context.Context) error { return errors.New("ping failed") },
			expectedErr:     errors.New("async database not initialized"),
			expectedSession: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := new(MockDB)
			mockDB.On("PingContext", mock.Anything).Return(tc.mockPing(mock.Anything))

			asyncDB = tc.expectedSession
			session, err := GetAsyncDB(context.Background())
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedSession, session)
			mockDB.AssertExpectations(t)
		})
	}
}