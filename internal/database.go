package database

import (
	"context"
	"database/sql"
	"os"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"internal/model"
)

var (
	once             sync.Once
	dbMutex          sync.Mutex
	asyncDB          *sqlx.DB
	syncDB           *sqlx.DB
	asyncSessionPool chan *sqlx.Tx
	syncSessionPool  chan *sqlx.Tx
)

func init() {
	once.Do(setupDatabase)
}

func setupDatabase() {
	var err error

	// Database configuration
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgresql://localhost/books"
	}

	// Create async engine for async operations
	asyncDB, err = sqlx.Open("postgres", databaseURL)
	if err != nil {
		panic(err)
	}

	// Create sync engine for sync operations (like creating tables)
	syncDB, err = sqlx.Open("postgres", databaseURL)
	if err != nil {
		panic(err)
	}

	// Initialize pools
	asyncSessionPool = make(chan *sqlx.Tx, 10)
	syncSessionPool = make(chan *sqlx.Tx, 10)

	// Pre-fill pools with transactions
	for i := 0; i < cap(asyncSessionPool); i++ {
		tx, err := asyncDB.BeginTxx(context.Background(), nil)
		if err != nil {
			panic(err)
		}
		asyncSessionPool <- tx
	}

	for i := 0; i < cap(syncSessionPool); i++ {
		tx, err := syncDB.BeginTxx(context.Background(), nil)
		if err != nil {
			panic(err)
		}
		syncSessionPool <- tx
	}
}

func InitModel(ctx context.Context) error {
	_, err := syncDB.ExecContext(ctx, model.CreateBooksTable())
	return err
}

func getAsyncSession(ctx context.Context) (*sqlx.Tx, error) {
	select {
	case tx := <-asyncSessionPool:
		return tx, nil
	default:
		return asyncDB.BeginTxx(ctx, nil)
	}
}

func returnAsyncSession(tx *sqlx.Tx) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if len(asyncSessionPool) < cap(asyncSessionPool) {
		asyncSessionPool <- tx
	}
}

func GetDB(ctx context.Context) (*sqlx.Tx, error) {
	session, err := getAsyncSession(ctx)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func CloseDB(ctx context.Context, session *sqlx.Tx) {
	if session != nil {
		returnAsyncSession(session)
	}
}

func getSyncSession(ctx context.Context) (*sqlx.Tx, error) {
	select {
	case tx := <-syncSessionPool:
		return tx, nil
	default:
		return syncDB.BeginTxx(ctx, nil)
	}
}

func returnSyncSession(tx *sqlx.Tx) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if len(syncSessionPool) < cap(syncSessionPool) {
		syncSessionPool <- tx
	}
}

func GetSyncDB(ctx context.Context) (*sqlx.Tx, error) {
	session, err := getSyncSession(ctx)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func CloseSyncDB(ctx context.Context, session *sqlx.Tx) {
	if session != nil {
		returnSyncSession(session)
	}
}