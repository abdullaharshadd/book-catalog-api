// internal/database/database_test.go
package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
)

type mockDB struct {
	mock.Mock
}

func (m *mockDB) Find(dest interface{}, wheres ...interface{}) *gorm.DB {
	args := m.Called(dest, wheres)
	return args.Get(0).(*gorm.DB)
}

func (m *mockDB) First(dest interface{}, wheres ...interface{}) *gorm.DB {
	args := m.Called(dest, wheres)
	return args.Get(0).(*gorm.DB)
}

func (m *mockDB) Create(value interface{}) *gorm.DB {
	args := m.Called(value)
	return args.Get(0).(*gorm.DB)
}

func (m *mockDB) Save(value interface{}) *gorm.DB {
	args := m.Called(value)
	return args.Get(0).(*gorm.DB)
}

func (m *mockDB) Delete(value interface{}) *gorm.DB {
	args := m.Called(value)
	return args.Get(0).(*gorm.DB)
}

type mockAsyncDB struct {
	mock.Mock
}

func (m *mockAsyncDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return m.Called(ctx, query, args).Get(0).(sql.Result), m.Called(ctx, query, args).Error(1)
}

func (m *mockAsyncDB) MustBegin() *sqlx.Tx {
	return m.Called().Get(0).(*sqlx.Tx)
}

type mockTx struct {
	mock.Mock
}

func (m *mockTx) Commit() error {
	return m.Called().Error(0)
}

func (m *mockTx) Rollback() error {
	return m.Called().Error(0)
}

func TestInitializeDatabase(t *testing.T) {
	tests := []struct {
		name       string
		setup      func()
		wantErr    bool
		sideEffect func()
	}{
		{
			name: "successful initialization",
			setup: func() {
				os.Setenv("DATABASE_URL", "test_dsn")
			},
			wantErr: false,
			sideEffect: func() {
				assert.NotNil(t, db)
				assert.NotNil(t, asyncDb)
			},
		},
		{
			name: "default initialization",
			setup: func() {
				os.Unsetenv("DATABASE_URL")
			},
			wantErr: false,
			sideEffect: func() {
				assert.NotNil(t, db)
				assert.NotNil(t, asyncDb)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer os.Unsetenv("DATABASE_URL")

			dbMock := &mockDB{}
			asyncDBMock := &mockAsyncDB{}

			dbMock.On("AutoMigrate", mock.Anything).Return(dbMock)
			asyncDBMock.On("ExecContext", mock.Anything, mock.Anything).Return(nil, nil)

			db = dbMock
			asyncDb = asyncDBMock

			err := InitializeDatabase()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.sideEffect()

			dbMock.AssertExpectations(t)
			asyncDBMock.AssertExpectations(t)
		})
	}
}

func TestCreateSchema(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(mockDB *mockAsyncDB)
		wantErr    bool
		sideEffect func(mockDB *mockAsyncDB)
	}{
		{
			name: "schema creation success",
			setup: func(mockDB *mockAsyncDB) {
				mockDB.On("ExecContext", mock.Anything, mock.Anything).Return(nil, nil)
			},
			wantErr: false,
			sideEffect: func(mockDB *mockAsyncDB) {
				mockDB.AssertCalled(t, "ExecContext", mock.Anything, mock.Anything)
			},
		},
		{
			name: "schema creation failure",
			setup: func(mockDB *mockAsyncDB) {
				mockDB.On("ExecContext", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("some error"))
			},
			wantErr: true,
			sideEffect: func(mockDB *mockAsyncDB) {
				mockDB.AssertCalled(t, "ExecContext", mock.Anything, mock.Anything)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockAsyncDB{}
			tt.setup(mockDB)

			asyncDb = mockDB

			err := CreateSchema(context.Background())
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.sideEffect(mockDB)
		})
	}
}

func TestListBooks(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(mockDB *mockDB)
		wantErr    bool
		sideEffect func(mockDB *mockDB)
	}{
		{
			name: "list books success",
			setup: func(mockDB *mockDB) {
				mockDB.On("Find", mock.Anything).Return(mockDB)
			},
			wantErr: false,
			sideEffect: func(mockDB *mockDB) {
				mockDB.AssertCalled(t, "Find", mock.Anything)
			},
		},
		{
			name: "list books failure",
			setup: func(mockDB *mockDB) {
				mockDB.On("Find", mock.Anything).Return(mockDB).Run(func(args mock.Arguments) {
					dbFind := args.Get(0).(*gorm.DB)
					dbFind.AddError(fmt.Errorf("some error"))
				})
			},
			wantErr: true,
			sideEffect: func(mockDB *mockDB) {
				mockDB.AssertCalled(t, "Find", mock.Anything)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDB{}
			tt.setup(mockDB)

			db = mockDB

			_, err := ListBooks(context.Background())
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.sideEffect(mockDB)
		})
	}
}

func TestGetBook(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(mockDB *mockDB)
		wantErr    bool
		sideEffect func(mockDB *mockDB)
	}{
		{
			name: "get book success",
			setup: func(mockDB *mockDB) {
				mockDB.On("First", mock.Anything, mock.Anything).Return(mockDB)
			},
			wantErr: false,
			sideEffect: func(mockDB *mockDB) {
				mockDB.AssertCalled(t, "First", mock.Anything, mock.Anything)
			},
		},
		{
			name: "get book not found",
			setup: func(mockDB *mockDB) {
				mockDB.On("First", mock.Anything, mock.Anything).Return(mockDB).Run(func(args mock.Arguments) {
					dbFirst := args.Get(0).(*gorm.DB)
					dbFirst.AddError(gorm.ErrRecordNotFound)
				})
			},
			wantErr: true,
			sideEffect: func(mockDB *mockDB) {
				mockDB.AssertCalled(t, "First", mock.Anything, mock.Anything)
			},
		},
		{
			name: "get book failure",
			setup: func(mockDB *mockDB) {
				mockDB.On("First", mock.Anything, mock.Anything).Return(mockDB).Run(func(args mock.Arguments) {
					dbFirst := args.Get(0).(*gorm.DB)
					dbFirst.AddError(fmt.Errorf("some error"))
				})
			},
			wantErr: true,
			sideEffect: func(mockDB *mockDB) {
				mockDB.AssertCalled(t, "First", mock.Anything, mock.Anything)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDB{}
			tt.setup(mockDB)

			db = mockDB

			_, err := GetBook(context.Background(), 1)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.sideEffect(mockDB)
		})
	}
}

func TestCreateBook(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(mockDB *mockDB)
		wantErr    bool
		sideEffect func(mockDB *mockDB)
	}{
		{
			name: "create book success",
			setup: func(mockDB *mockDB) {
				mockDB.On("Create", mock.Anything).Return(mockDB)
			},
			wantErr: false,
			sideEffect: func(mockDB *mockDB) {
				mockDB.AssertCalled(t, "Create", mock.Anything)
			},
		},
		{
			name: "create book validation failure",
			setup: func(mockDB *mockDB) {
				mockDB.On("Create", mock.Anything).Return(mockDB)
			},
			wantErr: true,
			sideEffect: func(mockDB *mockDB) {
				mockDB.AssertNotCalled(t, "Create", mock.Anything)
			},
		},
		{
			name: "create book failure",
			setup: func(mockDB *mockDB) {
				mockDB.On("Create", mock.Anything).Return(mockDB).Run(func(args mock.Arguments) {
					dbCreate := args.Get(0).(*gorm.DB)
					dbCreate.AddError(fmt.Errorf("some error"))
				})
			},
			wantErr: true,
			sideEffect: func(mockDB *mockDB) {
				mockDB.AssertCalled(t, "Create", mock.Anything)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDB{}
			tt.setup(mockDB)

			db = mockDB

			bookCreate := &schemas.BookCreate{
				Title:           "Test Book",
				Author:          "Test Author",
				PublishedYear:   2023,
				Summary:         "Test Summary",
				ValidationFunc:  func() error { return nil },
			}

			_, err := CreateBook(context.Background(), bookCreate)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.sideEffect(mockDB)
		})
	}
}

func TestUpdateBook(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(mockDB *mockDB)
		wantErr    bool
		sideEffect func(mockDB *mockDB)
	}{
		{
			name: "update book success",
			setup: func(mockDB *mockDB) {
				mockDB.On("First", mock.Anything, mock.Anything).Return(mockDB)
				mockDB.On("Save", mock.Anything).Return(mockDB)
			},
			wantErr: false,
			sideEffect: func(mockDB *mockDB) {
				mockDB.AssertCalled(t, "First", mock.Anything, mock.Anything)
				mockDB.AssertCalled(t, "Save", mock.Anything)
			},
		},
		{
			name: "update book validation failure",
			setup: func(mockDB *mockDB) {
				mockDB.On("First", mock.Anything, mock.Anything).Return(mockDB)
				mockDB.On("Save", mock.Anything).Return(mockDB)
			},
			wantErr: true,
			sideEffect: func(mockDB *mockDB) {
				mockDB.AssertCalled(t, "First", mock.Anything, mock.Anything)
				mockDB.AssertNotCalled(t, "Save", mock.Anything)
			},
		},
		{
			name: "update book failure",
			setup: func(mockDB *mockDB) {
				mockDB.On("First", mock.Anything, mock.Anything).Return(mockDB)
				mockDB.On("Save", mock.Anything).Return(mockDB).Run(func(args mock.Arguments) {
					dbSave := args.Get(0).(*gorm.DB)
					dbSave.AddError(fmt.Errorf("some error"))
				})
			},
			wantErr: true,
			sideEffect: func(mockDB *mockDB) {
				mockDB.AssertCalled(t, "First", mock.Anything, mock.Anything)
				mockDB.AssertCalled(t, "Save", mock.Anything)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDB{}
			tt.setup(mockDB)

			db = mockDB

			bookUpdate := &schemas.BookUpdate{
				Title:           "Updated Title",
				Author:          "Updated Author",
				PublishedYear:   2024,
				Summary:         "Updated Summary",
				ValidationFunc:  func() error { return nil },
			}

			_, err := UpdateBook(context.Background(), 1, bookUpdate)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.sideEffect(mockDB)
		})
	}
}

func TestDeleteBook(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(mockDB *mockDB)
		wantErr    bool
		sideEffect func(mockDB *mockDB)
	}{
		{
			name: "delete book success",
			setup: func(mockDB *mockDB) {
				mockDB.On("First", mock.Anything, mock.Anything).Return(mockDB)
				mockDB.On("Delete", mock.Anything).Return(mockDB)
			},
			wantErr: false,
			sideEffect: func(mockDB *mockDB) {
				mockDB.AssertCalled(t, "First", mock.Anything, mock.Anything)
				mockDB.AssertCalled(t, "Delete", mock.Anything)
			},
		},
		{
			name: "delete book failure",
			setup: func(mockDB *mockDB) {
				mockDB.On("First", mock.Anything, mock.Anything).Return(mockDB)
				mockDB.On("Delete", mock.Anything).Return(mockDB).Run(func(args mock.Arguments) {
					dbDelete := args.Get(0).(*gorm.DB)
					dbDelete.AddError(fmt.Errorf("some error"))
				})
			},
			wantErr: true,
			sideEffect: func(mockDB *mockDB) {
				mockDB.AssertCalled(t, "First", mock.Anything, mock.Anything)
				mockDB.AssertCalled(t, "Delete", mock.Anything)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDB{}
			tt.setup(mockDB)

			db = mockDB

			err := DeleteBook(context.Background(), 1)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.sideEffect(mockDB)
		})
	}
}

func TestNewTransaction(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(mockDB *mockAsyncDB)
		wantErr    bool
		sideEffect func(mockDB *mockAsyncDB)
	}{
		{
			name: "transaction success",
			setup: func(mockDB *mockAsyncDB) {
				mockDB.On("MustBegin").Return(&mockTx{})
			},
			wantErr: false,
			sideEffect: func(mockDB *mockAsyncDB) {
				mockDB.AssertCalled(t, "MustBegin")
			},
		},
		{
			name: "transaction failure",
			setup: func(mockDB *mockAsyncDB) {
				mockDB.On("MustBegin").Return(nil)
			},
			wantErr: true,
			sideEffect: func(mockDB *mockAsyncDB) {
				mockDB.AssertCalled(t, "MustBegin")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockAsyncDB{}
			tt.setup(mockDB)

			asyncDb = mockDB

			_, err := NewTransaction(context.Background())
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.sideEffect(mockDB)
		})
	}
}

func TestGetSession(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(mockAsyncDB *mockAsyncDB, mockTx *mockTx)
		wantErr    bool
		sideEffect func(mockAsyncDB *mockAsyncDB, mockTx *mockTx)
	}{
		{
			name: "session success",
			setup: func(mockAsyncDB *mockAsyncDB, mockTx *mockTx) {
				mockAsyncDB.On("MustBegin").Return(mockTx)
			},
			wantErr: false,
			sideEffect: func(mockAsyncDB *mockAsyncDB, mockTx *mockTx) {
				mockAsyncDB.AssertCalled(t, "MustBegin")
				mockTx.AssertNotCalled(t, "Commit")
				mockTx.AssertCalled(t, "Rollback")
			},
		},
		{
			name: "session failure",
			setup: func(mockAsyncDB *mockAsyncDB, mockTx *mockTx) {
				mockAsyncDB.On("MustBegin").Return(mockTx)
				mockTx.On("Commit").Return(fmt.Errorf("commit error"))
			},
			wantErr: true,
			sideEffect: func(mockAsyncDB *mockAsyncDB, mockTx *mockTx) {
				mockAsyncDB.AssertCalled(t, "MustBegin")
				mockTx.AssertCalled(t, "Commit")
				mockTx.AssertCalled(t, "Rollback")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAsyncDB := &mockAsyncDB{}
			mockTx := &mockTx{}
			tt.setup(mockAsyncDB, mockTx)

			asyncDb = mockAsyncDB

			_, err := GetSession(context.Background())
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.sideEffect(mockAsyncDB, mockTx)
		})
	}
}

func TestCloseSession(t *testing.T) {