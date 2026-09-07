package database

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSyncEngine is a mock implementation of the sync engine interface.
type MockSyncEngine struct {
	mock.Mock
}

func (m *MockSyncEngine) CreateTables() error {
	args := m.Called()
	return args.Error(0)
}

// MockAsyncSessionFactory is a mock implementation of the async session factory interface.
type MockAsyncSessionFactory struct {
	mock.Mock
}

func (m *MockAsyncSessionFactory) NewAsyncSession() (*AsyncSession, error) {
	args := m.Called()
	return args.Get(0).(*AsyncSession), args.Error(1)
}

// MockSyncSessionFactory is a mock implementation of the sync session factory interface.
type MockSyncSessionFactory struct {
	mock.Mock
}

func (m *MockSyncSessionFactory) NewSyncSession() (*SyncSession, error) {
	args := m.Called()
	return args.Get(0).(*SyncSession), args.Error(1)
}

// MockAsyncSession is a mock implementation of the async session interface.
type MockAsyncSession struct {
	mock.Mock
}

func (m *MockAsyncSession) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockAsyncSession) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

// MockSyncSession is a mock implementation of the sync session interface.
type MockSyncSession struct {
	mock.Mock
}

func (m *MockSyncSession) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSyncSession) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

var initDBTests = []struct {
	name              string
	syncEngineCreate  func() error
	expectedError     bool
	expectedSideEffect string
}{
	{"default_config", func() error { return nil }, false, "creates all tables in the database"},
	{"invalid_db_url", func() error { return ErrDBConnection }, true, ""},
}

func TestInitDB(t *testing.T) {
	for _, tt := range initDBTests {
		t.Run(tt.name, func(t *testing.T) {
			mockSyncEngine := new(MockSyncEngine)
			mockSyncEngine.On("CreateTables").Return(tt.syncEngineCreate())

			initDB(mockSyncEngine)

			if tt.expectedError {
				mockSyncEngine.AssertCalled(t, "CreateTables")
			} else {
				mockSyncEngine.AssertCalled(t, "CreateTables")
			}
		})
	}
}

var getDBTests = []struct {
	name           string
	sessionFactory func() (*AsyncSession, error)
	expectedOutput *AsyncSession
	expectedError  bool
}{
	{"valid_session", func() (*AsyncSession, error) { sess := &MockAsyncSession{}; return sess, nil }, &MockAsyncSession{}, false},
	{"invalid_session", func() (*AsyncSession, error) { return nil, ErrSessionCreation }, nil, true},
}

func TestGetDB(t *testing.T) {
	for _, tt := range getDBTests {
		t.Run(tt.name, func(t *testing.T) {
			mockSessionFactory := new(MockAsyncSessionFactory)
			mockSessionFactory.On("NewAsyncSession").Return(tt.sessionFactory())

			recorder := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			ctx := req.Context()

			session, err := getDB(ctx, mockSessionFactory)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, session)
				session.(*MockAsyncSession).On("Close").Return(nil)
				session.(*MockAsyncSession).On("Rollback").Return(nil)
			}

			mockSessionFactory.AssertCalled(t, "NewAsyncSession")
			session.(*MockAsyncSession).AssertCalled(t, "Close")
			if err != nil {
				session.(*MockAsyncSession).AssertCalled(t, "Rollback")
			}
		})
	}
}

var getSyncDBTests = []struct {
	name           string
	sessionFactory func() (*SyncSession, error)
	expectedOutput *SyncSession
	expectedError  bool
}{
	{"valid_sync_session", func() (*SyncSession, error) { sess := &MockSyncSession{}; return sess, nil }, &MockSyncSession{}, false},
	{"invalid_sync_session", func() (*SyncSession, error) { return nil, ErrSessionCreation }, nil, true},
}

func TestGetSyncDB(t *testing.T) {
	for _, tt := range getSyncDBTests {
		t.Run(tt.name, func(t *testing.T) {
			mockSessionFactory := new(MockSyncSessionFactory)
			mockSessionFactory.On("NewSyncSession").Return(tt.sessionFactory())

			session, err := getSyncDB(mockSessionFactory)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, session)
				session.(*MockSyncSession).On("Close").Return(nil)
				session.(*MockSyncSession).On("Rollback").Return(nil)
			}

			mockSessionFactory.AssertCalled(t, "NewSyncSession")
			session.(*MockSyncSession).AssertCalled(t, "Close")
			if err != nil {
				session.(*MockSyncSession).AssertCalled(t, "Rollback")
			}
		})
	}
}