package repository

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const testDBURL = "postgres://cqar:cqar_dev_password@localhost:5432/cqar_test?sslmode=disable"

// getTestDB returns a test database connection
func getTestDB(t *testing.T) *sql.DB {
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = testDBURL
	}

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err, "Failed to connect to test database")

	err = db.Ping()
	require.NoError(t, err, "Failed to ping test database")

	return db
}

// cleanupAssets removes test assets
func cleanupAssets(t *testing.T, db *sql.DB, symbols ...string) {
	if len(symbols) == 0 {
		return
	}

	query := "DELETE FROM assets WHERE symbol = ANY($1)"
	_, err := db.Exec(query, symbols)
	require.NoError(t, err, "Failed to cleanup test assets")
}
