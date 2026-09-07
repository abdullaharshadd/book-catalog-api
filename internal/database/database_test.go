package database

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"migrated-app/internal/model"
)

type mockGorm struct {
	mock.Mock
}

func (m *mockGorm) Open(dialect string, url string) (*gorm.DB, error) {
	args := m.Called(dialect, url)
	return args.Get(0).(*gorm.DB), args.Error(1)
}

func (m *mockGorm) AutoMigrate(v interface{}) *gorm.DB {
	m.Called(v)
	return m
}

func TestInitializeDatabase(t *testing.T) {
	type test struct {
		name              string
		databaseURL       string
		asyncDatabaseURL  string
		expectedErrOutput bool
	}

	tests := []test{
		{"default_urls", "", "", false},
		{"custom_urls", "sqlite3://memory:", "sqlite3://memory:", false},
		{"invalid_url", "invalid://url", "sqlite3://memory:", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := new(mockGorm)
			gorm.Open = mock.Open
			defer func() { gorm.Open = gorm.Open }()

			mock.On("Open", "postgres", tc.databaseURL).Return(&gorm.DB{}, nil)
			mock.On("Open", "postgres", tc.asyncDatabaseURL).Return(&gorm.DB{}, nil)
			mock.On("AutoMigrate", &model.Book{}).Return(&gorm.DB{})

			err := InitializeDatabase(context.Background())
			if tc.expectedErrOutput {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			mock.AssertExpectations(t)
		})
	}
}

func TestGetDb(t *testing.T) {
	type test struct {
		name        string
		dbInit      bool
		expectedErr bool
	}

	tests := []test{
		{"db_initialized", true, false},
		{"db_not_initialized", false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.dbInit {
				db, _ = gorm.Open("sqlite3", "file::memory:")
			} else {
				db = nil
			}

			session, err := GetDb(context.Background())
			if tc.expectedErr {
				assert.NotNil(t, err)
				assert.Nil(t, session)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, session)
			}
		})
	}
}

func TestGetAsyncDb(t *testing.T) {
	type test struct {
		name        string
		asyncDbInit bool
		expectedErr bool
	}

	tests := []test{
		{"async_db_initialized", true, false},
		{"async_db_not_initialized", false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.asyncDbInit {
				asyncDb, _ = gorm.Open("sqlite3", "file::memory:")
			} else {
				asyncDb = nil
			}

			session, err := GetAsyncDb(context.Background())
			if tc.expectedErr {
				assert.NotNil(t, err)
				assert.Nil(t, session)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, session)
			}
		})
	}
}

func TestGetDbHTTPHandler(t *testing.T) {
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		session, err := GetDb(r.Context())
		assert.Nil(t, err)
		assert.NotNil(t, session)
		session.Close()
	}

	server := httptest.NewServer(http.HandlerFunc(httpHandler))
	defer server.Close()

	resp, err := http.Get(server.URL)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetAsyncDbHTTPHandler(t *testing.T) {
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		session, err := GetAsyncDb(r.Context())
		assert.Nil(t, err)
		assert.NotNil(t, session)
		session.Close()
	}

	server := httptest.NewServer(http.HandlerFunc(httpHandler))
	defer server.Close()

	resp, err := http.Get(server.URL)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}