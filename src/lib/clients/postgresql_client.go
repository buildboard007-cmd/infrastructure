package clients

import (
	"database/sql"
	"fmt"
	"infrastructure/lib/constants"

	_ "github.com/lib/pq"
)

// NewPostgresSQLClient creates a new PostgreSQL client with connection pooling optimized for Lambda
func NewPostgresSQLClient(host, port, dbname, user, password, sslMode string) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslMode,
	)

	fmt.Println("connection string: ", connStr)

	db, err := sql.Open(constants.DRIVER_NAME, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open db connection: %w", err)
	}

	// Lambda-optimized connection settings
	db.SetMaxOpenConns(2) // Max 2 open connections for Lambda
	db.SetMaxIdleConns(1) // Keep 1 idle connection

	// Validate connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
