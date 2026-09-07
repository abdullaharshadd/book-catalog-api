package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"migrated-app/internal/model"
)

type mockSQLX struct {
	mock.Mock
}

func (m *mockSQLX) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return m.Called(ctx, query, args).Get(0), m.Called(ctx, query, args).Error(1)
}

func (m *mockSQLX) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	return m.Called(ctx, opts).Get(0).(*sqlx.Tx), m.Called(ctx, opts).Error(1)
}

func TestInitModel(t *testing.T) {
	type args struct{}
	type fields struct {
		syncDB *mockSQLX
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "successful initialization",
			args: args{},
			fields: fields{
				syncDB: &mockSQLX{},
			},
			wantErr: false,
		},
		{
			name: "failed initialization due to exec error",
			args: args{},
			fields: fields{
				syncDB: &mockSQLX{Mock: mock.Mock{
					On("ExecContext", mock.Anything, model.CreateBooksTable(), mock.Anything).
						Return(nil, assert.AnError),
				}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncDB = tt.fields.syncDB
			err := InitModel(context.Background())
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			tt.fields.syncDB.AssertExpectations(t)
		})
	}
}

func TestGetDB(t *testing.T) {
	type fields struct {
		asyncDB *mockSQLX
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "successful transaction retrieval",
			fields: fields{
				asyncDB: &mockSQLX{Mock: mock.Mock{
					On("BeginTxx", mock.Anything, mock.Anything).
						Return(&sqlx.Tx{}, nil),
				}},
			},
			wantErr: false,
		},
		{
			name: "failed transaction retrieval",
			fields: fields{
				asyncDB: &mockSQLX{Mock: mock.Mock{
					On("BeginTxx", mock.Anything, mock.Anything).
						Return(nil, assert.AnError),
				}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asyncDB = tt.fields.asyncDB
			got, err := GetDB(context.Background())
			if tt.wantErr {
				assert.Nil(t, got)
				require.Error(t, err)
			} else {
				require.NotNil(t, got)
				require.NoError(t, err)
			}
			tt.fields.asyncDB.AssertExpectations(t)
		})
	}
}

func TestCloseDB(t *testing.T) {
	type args struct {
		ctx  context.Context
		session *sqlx.Tx
	}
	tests := []struct {
		name    string
		args    args
	}{
		{
			name: "close valid transaction",
			args: args{
				ctx:      context.Background(),
				session:  &sqlx.Tx{},
			},
		},
		{
			name: "close nil transaction",
			args: args{
				ctx:      context.Background(),
				session:  nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CloseDB(tt.args.ctx, tt.args.session)
		})
	}
}

func TestGetSyncDB(t *testing.T) {
	type fields struct {
		syncDB *mockSQLX
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "successful transaction retrieval",
			fields: fields{
				syncDB: &mockSQLX{Mock: mock.Mock{
					On("BeginTxx", mock.Anything, mock.Anything).
						Return(&sqlx.Tx{}, nil),
				}},
			},
			wantErr: false,
		},
		{
			name: "failed transaction retrieval",
			fields: fields{
				syncDB: &mockSQLX{Mock: mock.Mock{
					On("BeginTxx", mock.Anything, mock.Anything).
						Return(nil, assert.AnError),
				}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncDB = tt.fields.syncDB
			got, err := GetSyncDB(context.Background())
			if tt.wantErr {
				assert.Nil(t, got)
				require.Error(t, err)
			} else {
				require.NotNil(t, got)
				require.NoError(t, err)
			}
			tt.fields.syncDB.AssertExpectations(t)
		})
	}
}

func TestCloseSyncDB(t *testing.T) {
	type args struct {
		ctx  context.Context
		session *sqlx.Tx
	}
	tests := []struct {
		name    string
		args    args
	}{
		{
			name: "close valid transaction",
			args: args{
				ctx:      context.Background(),
				session:  &sqlx.Tx{},
			},
		},
		{
			name: "close nil transaction",
			args: args{
				ctx:      context.Background(),
				session:  nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CloseSyncDB(tt.args.ctx, tt.args.session)
		})
	}
}