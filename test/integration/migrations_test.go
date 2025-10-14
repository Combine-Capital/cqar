package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultDBURL = "postgres://cqar:cqar_dev_password@localhost:5432/cqar_test?sslmode=disable"
)

func getTestDB(t *testing.T) *sql.DB {
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = defaultDBURL
	}

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err, "Failed to connect to test database")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	require.NoError(t, err, "Failed to ping test database")

	return db
}

func TestMigrations_AssetsTable(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	t.Run("table exists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'assets'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "assets table should exist")
	})

	t.Run("has required columns", func(t *testing.T) {
		rows, err := db.Query(`
			SELECT column_name, data_type, is_nullable
			FROM information_schema.columns
			WHERE table_name = 'assets'
			ORDER BY ordinal_position
		`)
		require.NoError(t, err)
		defer rows.Close()

		expectedColumns := map[string]struct {
			dataType   string
			isNullable string
		}{
			"id":          {"uuid", "NO"},
			"symbol":      {"character varying", "NO"},
			"name":        {"character varying", "NO"},
			"type":        {"character varying", "NO"},
			"category":    {"character varying", "YES"},
			"description": {"text", "YES"},
			"logo_url":    {"text", "YES"},
			"website_url": {"text", "YES"},
			"created_at":  {"timestamp with time zone", "NO"},
			"updated_at":  {"timestamp with time zone", "NO"},
		}

		foundColumns := make(map[string]bool)
		for rows.Next() {
			var colName, dataType, isNullable string
			require.NoError(t, rows.Scan(&colName, &dataType, &isNullable))

			expected, exists := expectedColumns[colName]
			assert.True(t, exists, "Unexpected column: %s", colName)
			if exists {
				assert.Equal(t, expected.dataType, dataType, "Wrong data type for column %s", colName)
				assert.Equal(t, expected.isNullable, isNullable, "Wrong nullable constraint for column %s", colName)
			}
			foundColumns[colName] = true
		}

		for colName := range expectedColumns {
			assert.True(t, foundColumns[colName], "Missing column: %s", colName)
		}
	})

	t.Run("has required indexes", func(t *testing.T) {
		expectedIndexes := []string{
			"idx_assets_symbol",
			"idx_assets_type",
			"idx_assets_created_at",
		}

		for _, indexName := range expectedIndexes {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM pg_indexes
					WHERE schemaname = 'public'
					AND tablename = 'assets'
					AND indexname = $1
				)
			`, indexName).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Index %s should exist", indexName)
		}
	})

	t.Run("can insert and query asset", func(t *testing.T) {
		assetID := uuid.New()
		now := time.Now().UTC().Truncate(time.Microsecond)

		_, err := db.Exec(`
			INSERT INTO assets (id, symbol, name, type, category, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, assetID, "TEST", "Test Asset", "CRYPTOCURRENCY", "Testing", now, now)
		require.NoError(t, err)

		var symbol, name string
		err = db.QueryRow(`SELECT symbol, name FROM assets WHERE id = $1`, assetID).Scan(&symbol, &name)
		require.NoError(t, err)
		assert.Equal(t, "TEST", symbol)
		assert.Equal(t, "Test Asset", name)

		// Cleanup
		_, err = db.Exec(`DELETE FROM assets WHERE id = $1`, assetID)
		require.NoError(t, err)
	})
}

func TestMigrations_SymbolsTable(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Setup: Create test assets
	baseAssetID := uuid.New()
	quoteAssetID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	_, err := db.Exec(`
		INSERT INTO assets (id, symbol, name, type, created_at, updated_at)
		VALUES 
			($1, 'BASE', 'Base Asset', 'CRYPTOCURRENCY', $3, $3),
			($2, 'QUOTE', 'Quote Asset', 'STABLECOIN', $3, $3)
	`, baseAssetID, quoteAssetID, now)
	require.NoError(t, err)
	defer func() {
		db.Exec(`DELETE FROM symbols WHERE base_asset_id IN ($1, $2)`, baseAssetID, quoteAssetID)
		db.Exec(`DELETE FROM assets WHERE id IN ($1, $2)`, baseAssetID, quoteAssetID)
	}()

	t.Run("table exists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'symbols'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "symbols table should exist")
	})

	t.Run("has required indexes", func(t *testing.T) {
		expectedIndexes := []string{
			"idx_symbols_base_asset",
			"idx_symbols_quote_asset",
			"idx_symbols_type",
			"idx_symbols_base_quote",
			"idx_symbols_created_at",
		}

		for _, indexName := range expectedIndexes {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM pg_indexes
					WHERE schemaname = 'public'
					AND tablename = 'symbols'
					AND indexname = $1
				)
			`, indexName).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Index %s should exist", indexName)
		}
	})

	t.Run("enforces foreign key constraints", func(t *testing.T) {
		symbolID := uuid.New()
		invalidAssetID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO symbols (id, base_asset_id, quote_asset_id, symbol_type, 
								tick_size, lot_size, min_order_size, max_order_size, created_at, updated_at)
			VALUES ($1, $2, $3, 'SPOT', 0.01, 0.001, 0.001, 1000, $4, $4)
		`, symbolID, invalidAssetID, quoteAssetID, now)

		assert.Error(t, err, "Should fail with invalid base_asset_id")
		assert.Contains(t, err.Error(), "foreign key constraint", "Error should mention foreign key constraint")
	})

	t.Run("enforces check constraints", func(t *testing.T) {
		symbolID := uuid.New()

		// Test negative tick_size
		_, err := db.Exec(`
			INSERT INTO symbols (id, base_asset_id, quote_asset_id, symbol_type, 
								tick_size, lot_size, min_order_size, max_order_size, created_at, updated_at)
			VALUES ($1, $2, $3, 'SPOT', -0.01, 0.001, 0.001, 1000, $4, $4)
		`, symbolID, baseAssetID, quoteAssetID, now)
		assert.Error(t, err, "Should fail with negative tick_size")
		assert.Contains(t, err.Error(), "chk_tick_size_positive", "Error should mention tick_size constraint")

		// Test min > max order size
		_, err = db.Exec(`
			INSERT INTO symbols (id, base_asset_id, quote_asset_id, symbol_type, 
								tick_size, lot_size, min_order_size, max_order_size, created_at, updated_at)
			VALUES ($1, $2, $3, 'SPOT', 0.01, 0.001, 1000, 100, $4, $4)
		`, symbolID, baseAssetID, quoteAssetID, now)
		assert.Error(t, err, "Should fail with min_order_size > max_order_size")
		assert.Contains(t, err.Error(), "chk_order_size_range", "Error should mention order size constraint")
	})

	t.Run("enforces unique constraint for options", func(t *testing.T) {
		symbolID1 := uuid.New()
		symbolID2 := uuid.New()
		expiry := time.Now().UTC().Add(30 * 24 * time.Hour).Truncate(time.Microsecond)

		// Insert first option symbol
		_, err := db.Exec(`
			INSERT INTO symbols (id, base_asset_id, quote_asset_id, symbol_type, 
								tick_size, lot_size, min_order_size, max_order_size,
								strike_price, expiry, option_type, created_at, updated_at)
			VALUES ($1, $2, $3, 'OPTION', 0.01, 0.001, 0.001, 1000, 50000, $4, 'CALL', $5, $5)
		`, symbolID1, baseAssetID, quoteAssetID, expiry, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM symbols WHERE id = $1`, symbolID1)

		// Try to insert duplicate option symbol (same base/quote/strike/expiry)
		_, err = db.Exec(`
			INSERT INTO symbols (id, base_asset_id, quote_asset_id, symbol_type, 
								tick_size, lot_size, min_order_size, max_order_size,
								strike_price, expiry, option_type, created_at, updated_at)
			VALUES ($1, $2, $3, 'OPTION', 0.01, 0.001, 0.001, 1000, 50000, $4, 'CALL', $5, $5)
		`, symbolID2, baseAssetID, quoteAssetID, expiry, now)
		assert.Error(t, err, "Should fail with duplicate option symbol")
		assert.Contains(t, err.Error(), "unique_symbol", "Error should mention unique constraint")
	})

	t.Run("can insert and query spot symbol", func(t *testing.T) {
		symbolID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO symbols (id, base_asset_id, quote_asset_id, symbol_type, 
								tick_size, lot_size, min_order_size, max_order_size, created_at, updated_at)
			VALUES ($1, $2, $3, 'SPOT', 0.01, 0.001, 0.001, 1000, $4, $4)
		`, symbolID, baseAssetID, quoteAssetID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM symbols WHERE id = $1`, symbolID)

		var symbolType string
		var tickSize float64
		err = db.QueryRow(`SELECT symbol_type, tick_size FROM symbols WHERE id = $1`, symbolID).Scan(&symbolType, &tickSize)
		require.NoError(t, err)
		assert.Equal(t, "SPOT", symbolType)
		assert.Equal(t, 0.01, tickSize)
	})

	t.Run("can insert option symbol with required fields", func(t *testing.T) {
		symbolID := uuid.New()
		expiry := time.Now().UTC().Add(30 * 24 * time.Hour).Truncate(time.Microsecond)

		_, err := db.Exec(`
			INSERT INTO symbols (id, base_asset_id, quote_asset_id, symbol_type, 
								tick_size, lot_size, min_order_size, max_order_size,
								strike_price, expiry, option_type, created_at, updated_at)
			VALUES ($1, $2, $3, 'OPTION', 0.01, 0.001, 0.001, 1000, 50000, $4, 'CALL', $5, $5)
		`, symbolID, baseAssetID, quoteAssetID, expiry, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM symbols WHERE id = $1`, symbolID)

		var optionType string
		var strikePrice float64
		err = db.QueryRow(`SELECT option_type, strike_price FROM symbols WHERE id = $1`, symbolID).Scan(&optionType, &strikePrice)
		require.NoError(t, err)
		assert.Equal(t, "CALL", optionType)
		assert.Equal(t, 50000.0, strikePrice)
	})

	t.Run("enforces option fields constraint", func(t *testing.T) {
		symbolID := uuid.New()

		// Try to insert OPTION without required fields
		_, err := db.Exec(`
			INSERT INTO symbols (id, base_asset_id, quote_asset_id, symbol_type, 
								tick_size, lot_size, min_order_size, max_order_size, created_at, updated_at)
			VALUES ($1, $2, $3, 'OPTION', 0.01, 0.001, 0.001, 1000, $4, $4)
		`, symbolID, baseAssetID, quoteAssetID, now)
		assert.Error(t, err, "Should fail when OPTION missing strike/expiry/type")
		assert.Contains(t, err.Error(), "chk_option_fields", "Error should mention option fields constraint")
	})
}

func TestMigrations_ChainsTable(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	t.Run("table exists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'chains'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "chains table should exist")
	})

	t.Run("has required indexes", func(t *testing.T) {
		expectedIndexes := []string{
			"idx_chains_type",
			"idx_chains_native_asset",
		}

		for _, indexName := range expectedIndexes {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM pg_indexes
					WHERE schemaname = 'public'
					AND tablename = 'chains'
					AND indexname = $1
				)
			`, indexName).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Index %s should exist", indexName)
		}
	})

	t.Run("enforces chain_id format constraint", func(t *testing.T) {
		// Try invalid chain_id with uppercase
		_, err := db.Exec(`
			INSERT INTO chains (id, name, chain_type, created_at)
			VALUES ('InvalidChain', 'Invalid Chain', 'EVM', $1)
		`, time.Now())
		assert.Error(t, err, "Should fail with uppercase chain_id")
		assert.Contains(t, err.Error(), "chk_chain_id_format", "Error should mention chain_id format constraint")

		// Try invalid chain_id with spaces
		_, err = db.Exec(`
			INSERT INTO chains (id, name, chain_type, created_at)
			VALUES ('invalid chain', 'Invalid Chain', 'EVM', $1)
		`, time.Now())
		assert.Error(t, err, "Should fail with spaces in chain_id")
	})

	t.Run("can insert and query chain", func(t *testing.T) {
		chainID := "test_chain"
		rpcURLs := []string{"https://rpc1.example.com", "https://rpc2.example.com"}

		_, err := db.Exec(`
			INSERT INTO chains (id, name, chain_type, rpc_urls, explorer_url, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, chainID, "Test Chain", "EVM", fmt.Sprintf("{%s}", "\""+rpcURLs[0]+"\",\""+rpcURLs[1]+"\""), "https://explorer.example.com", time.Now())
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM chains WHERE id = $1`, chainID)

		var name, chainType string
		err = db.QueryRow(`SELECT name, chain_type FROM chains WHERE id = $1`, chainID).Scan(&name, &chainType)
		require.NoError(t, err)
		assert.Equal(t, "Test Chain", name)
		assert.Equal(t, "EVM", chainType)
	})
}

func TestMigrations_VenuesTable(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Setup: Create test chain
	chainID := "test_venue_chain"
	_, err := db.Exec(`
		INSERT INTO chains (id, name, chain_type, created_at)
		VALUES ($1, 'Test Venue Chain', 'EVM', $2)
	`, chainID, time.Now())
	require.NoError(t, err)
	defer db.Exec(`DELETE FROM chains WHERE id = $1`, chainID)

	t.Run("table exists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'venues'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "venues table should exist")
	})

	t.Run("has required indexes", func(t *testing.T) {
		expectedIndexes := []string{
			"idx_venues_type",
			"idx_venues_chain_id",
			"idx_venues_active",
		}

		for _, indexName := range expectedIndexes {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM pg_indexes
					WHERE schemaname = 'public'
					AND tablename = 'venues'
					AND indexname = $1
				)
			`, indexName).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Index %s should exist", indexName)
		}
	})

	t.Run("enforces venue_id format constraint", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO venues (id, name, venue_type, created_at)
			VALUES ('Invalid Venue', 'Invalid', 'CEX', $1)
		`, time.Now())
		assert.Error(t, err, "Should fail with spaces in venue_id")
		assert.Contains(t, err.Error(), "chk_venue_id_format", "Error should mention venue_id format constraint")
	})

	t.Run("enforces venue_type enum", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO venues (id, name, venue_type, chain_id, created_at)
			VALUES ('test_venue', 'Test', 'INVALID_TYPE', $1, $2)
		`, chainID, time.Now())
		assert.Error(t, err, "Should fail with invalid venue_type")
		assert.Contains(t, err.Error(), "chk_venue_type", "Error should mention venue_type constraint")
	})

	t.Run("enforces DEX requires chain_id", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO venues (id, name, venue_type, created_at)
			VALUES ('test_dex', 'Test DEX', 'DEX', $1)
		`, time.Now())
		assert.Error(t, err, "Should fail when DEX missing chain_id")
		assert.Contains(t, err.Error(), "chk_dex_has_chain", "Error should mention DEX chain constraint")
	})

	t.Run("can insert CEX without chain_id", func(t *testing.T) {
		venueID := "test_cex"

		_, err := db.Exec(`
			INSERT INTO venues (id, name, venue_type, api_endpoint, is_active, created_at)
			VALUES ($1, 'Test CEX', 'CEX', 'https://api.test.com', true, $2)
		`, venueID, time.Now())
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM venues WHERE id = $1`, venueID)

		var name, venueType string
		err = db.QueryRow(`SELECT name, venue_type FROM venues WHERE id = $1`, venueID).Scan(&name, &venueType)
		require.NoError(t, err)
		assert.Equal(t, "Test CEX", name)
		assert.Equal(t, "CEX", venueType)
	})

	t.Run("can insert DEX with chain_id", func(t *testing.T) {
		venueID := "test_dex"

		_, err := db.Exec(`
			INSERT INTO venues (id, name, venue_type, chain_id, protocol_address, is_active, created_at)
			VALUES ($1, 'Test DEX', 'DEX', $2, '0x1234567890abcdef', true, $3)
		`, venueID, chainID, time.Now())
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM venues WHERE id = $1`, venueID)

		var name string
		var chain sql.NullString
		err = db.QueryRow(`SELECT name, chain_id FROM venues WHERE id = $1`, venueID).Scan(&name, &chain)
		require.NoError(t, err)
		assert.Equal(t, "Test DEX", name)
		assert.True(t, chain.Valid)
		assert.Equal(t, chainID, chain.String)
	})
}
