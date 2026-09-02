package internal

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/spf13/viper"
	"github.com/rs/zerolog/log"
)

var (
	databaseURL = viper.GetString("DATABASE_URL")
	asyncDatabaseURL = viper.GetString("ASYNC_DATABASE_URL")
)

// Database provides a way to interact with the database.
// It encapsulates the database connection and provides methods to create sessions.
type Database struct {
	syncDB  *sqlx.DB
	asyncDB *sqlx.DB
}

// NewDatabase initializes a new Database instance.
func NewDatabase() (*Database, error) {
	syncDB, err := sqlx.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	asyncDB, err := sqlx.Open("postgres", asyncDatabaseURL)
	if err != nil {
		return nil, err
	}
	return &Database{syncDB: syncDB, asyncDB: asyncDB}, nil
}

// InitDB initializes the database schema.
func (db *Database) InitDB() error {
	// Replace this with actual DDL SQL statements or migration files
	ddl := `CREATE TABLE IF NOT EXISTS books (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		author VARCHAR(255) NOT NULL
	);`
	_, err := db.syncDB.Exec(ddl)
	if err != nil {
		return err
	}
	return nil
}

// GetDB provides a new async database session.
func (db *Database) GetDB(ctx context.Context) (*sqlx.Tx, error) {
	tx := db.asyncDB.MustBeginTxx(ctx, &sql.TxOptions{})
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	return tx, nil
}

// GetSyncDB provides a new sync database session.
func (db *Database) GetSyncDB() (*sqlx.Tx, error) {
	tx := db.syncDB.MustBegin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	return tx, nil
}
