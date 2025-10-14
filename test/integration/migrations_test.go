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

func TestMigrations_DeploymentsTable(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Setup: Create test asset and chain
	assetID := uuid.New()
	chainID := "test_deploy_chain"
	now := time.Now().UTC().Truncate(time.Microsecond)

	_, err := db.Exec(`
		INSERT INTO assets (id, symbol, name, type, created_at, updated_at)
		VALUES ($1, 'USDC', 'USD Coin', 'STABLECOIN', $2, $2)
	`, assetID, now)
	require.NoError(t, err)
	defer db.Exec(`DELETE FROM assets WHERE id = $1`, assetID)

	_, err = db.Exec(`
		INSERT INTO chains (id, name, chain_type, created_at)
		VALUES ($1, 'Test Deploy Chain', 'EVM', $2)
	`, chainID, now)
	require.NoError(t, err)
	defer db.Exec(`DELETE FROM chains WHERE id = $1`, chainID)

	t.Run("table exists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'deployments'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "deployments table should exist")
	})

	t.Run("has required indexes", func(t *testing.T) {
		expectedIndexes := []string{
			"idx_deployments_asset_id",
			"idx_deployments_chain_id",
			"idx_deployments_canonical",
			"idx_deployments_asset_chain",
			"idx_deployments_contract_address",
		}

		for _, indexName := range expectedIndexes {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM pg_indexes
					WHERE schemaname = 'public'
					AND tablename = 'deployments'
					AND indexname = $1
				)
			`, indexName).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Index %s should exist", indexName)
		}
	})

	t.Run("enforces foreign key constraints", func(t *testing.T) {
		deploymentID := uuid.New()
		invalidAssetID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO deployments (id, asset_id, chain_id, contract_address, decimals, created_at, updated_at)
			VALUES ($1, $2, $3, '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', 6, $4, $4)
		`, deploymentID, invalidAssetID, chainID, now)

		assert.Error(t, err, "Should fail with invalid asset_id")
		assert.Contains(t, err.Error(), "foreign key constraint", "Error should mention foreign key constraint")
	})

	t.Run("enforces unique asset+chain constraint", func(t *testing.T) {
		deploymentID1 := uuid.New()
		deploymentID2 := uuid.New()

		_, err := db.Exec(`
			INSERT INTO deployments (id, asset_id, chain_id, contract_address, decimals, created_at, updated_at)
			VALUES ($1, $2, $3, '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', 6, $4, $4)
		`, deploymentID1, assetID, chainID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM deployments WHERE id = $1`, deploymentID1)

		// Try to insert duplicate asset+chain
		_, err = db.Exec(`
			INSERT INTO deployments (id, asset_id, chain_id, contract_address, decimals, created_at, updated_at)
			VALUES ($1, $2, $3, '0xdifferentaddress', 6, $4, $4)
		`, deploymentID2, assetID, chainID, now)
		assert.Error(t, err, "Should fail with duplicate asset+chain")
		assert.Contains(t, err.Error(), "unique_asset_chain_deployment", "Error should mention unique constraint")
	})

	t.Run("enforces decimals range constraint", func(t *testing.T) {
		deploymentID := uuid.New()
		chainID2 := "test_chain2"

		_, err := db.Exec(`INSERT INTO chains (id, name, chain_type, created_at) VALUES ($1, 'Test Chain 2', 'EVM', $2)`, chainID2, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM chains WHERE id = $1`, chainID2)

		_, err = db.Exec(`
			INSERT INTO deployments (id, asset_id, chain_id, contract_address, decimals, created_at, updated_at)
			VALUES ($1, $2, $3, '0xtest', 100, $4, $4)
		`, deploymentID, assetID, chainID2, now)
		assert.Error(t, err, "Should fail with decimals > 77")
		assert.Contains(t, err.Error(), "chk_decimals_range", "Error should mention decimals constraint")
	})

	t.Run("can insert and query deployment", func(t *testing.T) {
		deploymentID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO deployments (id, asset_id, chain_id, contract_address, decimals, is_canonical, created_at, updated_at)
			VALUES ($1, $2, $3, '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', 6, true, $4, $4)
		`, deploymentID, assetID, chainID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM deployments WHERE id = $1`, deploymentID)

		var contractAddress string
		var decimals int
		var isCanonical bool
		err = db.QueryRow(`SELECT contract_address, decimals, is_canonical FROM deployments WHERE id = $1`, deploymentID).
			Scan(&contractAddress, &decimals, &isCanonical)
		require.NoError(t, err)
		assert.Equal(t, "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", contractAddress)
		assert.Equal(t, 6, decimals)
		assert.True(t, isCanonical)
	})
}

func TestMigrations_RelationshipsTable(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Setup: Create test assets
	ethID := uuid.New()
	wethID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	_, err := db.Exec(`
		INSERT INTO assets (id, symbol, name, type, created_at, updated_at)
		VALUES 
			($1, 'ETH', 'Ethereum', 'CRYPTOCURRENCY', $3, $3),
			($2, 'WETH', 'Wrapped ETH', 'WRAPPED', $3, $3)
	`, ethID, wethID, now)
	require.NoError(t, err)
	defer func() {
		db.Exec(`DELETE FROM relationships WHERE from_asset_id IN ($1, $2) OR to_asset_id IN ($1, $2)`, ethID, wethID)
		db.Exec(`DELETE FROM assets WHERE id IN ($1, $2)`, ethID, wethID)
	}()

	t.Run("table exists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'relationships'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "relationships table should exist")
	})

	t.Run("has required indexes", func(t *testing.T) {
		expectedIndexes := []string{
			"idx_relationships_from_asset",
			"idx_relationships_to_asset",
			"idx_relationships_type",
			"idx_relationships_bidirectional",
		}

		for _, indexName := range expectedIndexes {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM pg_indexes
					WHERE schemaname = 'public'
					AND tablename = 'relationships'
					AND indexname = $1
				)
			`, indexName).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Index %s should exist", indexName)
		}
	})

	t.Run("enforces relationship_type enum", func(t *testing.T) {
		relationshipID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO relationships (id, from_asset_id, to_asset_id, relationship_type, created_at, updated_at)
			VALUES ($1, $2, $3, 'INVALID_TYPE', $4, $4)
		`, relationshipID, wethID, ethID, now)
		assert.Error(t, err, "Should fail with invalid relationship_type")
		assert.Contains(t, err.Error(), "chk_relationship_type", "Error should mention relationship_type constraint")
	})

	t.Run("prevents self-referential relationships", func(t *testing.T) {
		relationshipID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO relationships (id, from_asset_id, to_asset_id, relationship_type, created_at, updated_at)
			VALUES ($1, $2, $2, 'WRAPS', $3, $3)
		`, relationshipID, ethID, now)
		assert.Error(t, err, "Should fail with self-referential relationship")
		assert.Contains(t, err.Error(), "chk_no_self_reference", "Error should mention self-reference constraint")
	})

	t.Run("can insert and query relationship", func(t *testing.T) {
		relationshipID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO relationships (id, from_asset_id, to_asset_id, relationship_type, conversion_rate, protocol, created_at, updated_at)
			VALUES ($1, $2, $3, 'WRAPS', 1.0, 'WETH9', $4, $4)
		`, relationshipID, wethID, ethID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM relationships WHERE id = $1`, relationshipID)

		var relType, protocol string
		var conversionRate float64
		err = db.QueryRow(`SELECT relationship_type, conversion_rate, protocol FROM relationships WHERE id = $1`, relationshipID).
			Scan(&relType, &conversionRate, &protocol)
		require.NoError(t, err)
		assert.Equal(t, "WRAPS", relType)
		assert.Equal(t, 1.0, conversionRate)
		assert.Equal(t, "WETH9", protocol)
	})
}

func TestMigrations_QualityFlagsTable(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Setup: Create test asset
	assetID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	_, err := db.Exec(`
		INSERT INTO assets (id, symbol, name, type, created_at, updated_at)
		VALUES ($1, 'SCAM', 'Scam Token', 'CRYPTOCURRENCY', $2, $2)
	`, assetID, now)
	require.NoError(t, err)
	defer func() {
		db.Exec(`DELETE FROM quality_flags WHERE asset_id = $1`, assetID)
		db.Exec(`DELETE FROM assets WHERE id = $1`, assetID)
	}()

	t.Run("table exists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'quality_flags'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "quality_flags table should exist")
	})

	t.Run("has required indexes", func(t *testing.T) {
		expectedIndexes := []string{
			"idx_quality_flags_asset_id",
			"idx_quality_flags_type",
			"idx_quality_flags_severity",
			"idx_quality_flags_active_critical",
			"idx_quality_flags_active",
		}

		for _, indexName := range expectedIndexes {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM pg_indexes
					WHERE schemaname = 'public'
					AND tablename = 'quality_flags'
					AND indexname = $1
				)
			`, indexName).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Index %s should exist", indexName)
		}
	})

	t.Run("enforces flag_type enum", func(t *testing.T) {
		flagID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO quality_flags (id, asset_id, flag_type, severity, source, reason, raised_at)
			VALUES ($1, $2, 'INVALID_TYPE', 'CRITICAL', 'manual', 'test', $3)
		`, flagID, assetID, now)
		assert.Error(t, err, "Should fail with invalid flag_type")
		assert.Contains(t, err.Error(), "chk_flag_type", "Error should mention flag_type constraint")
	})

	t.Run("enforces severity enum", func(t *testing.T) {
		flagID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO quality_flags (id, asset_id, flag_type, severity, source, reason, raised_at)
			VALUES ($1, $2, 'SCAM', 'INVALID_SEVERITY', 'manual', 'test', $3)
		`, flagID, assetID, now)
		assert.Error(t, err, "Should fail with invalid severity")
		assert.Contains(t, err.Error(), "chk_severity", "Error should mention severity constraint")
	})

	t.Run("can insert and query quality flag", func(t *testing.T) {
		flagID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO quality_flags (id, asset_id, flag_type, severity, source, reason, evidence_url, raised_at)
			VALUES ($1, $2, 'SCAM', 'CRITICAL', 'automated_scanner', 'Contract has rug pull indicators', 'https://example.com/evidence', $3)
		`, flagID, assetID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM quality_flags WHERE id = $1`, flagID)

		var flagType, severity, source, reason string
		err = db.QueryRow(`SELECT flag_type, severity, source, reason FROM quality_flags WHERE id = $1`, flagID).
			Scan(&flagType, &severity, &source, &reason)
		require.NoError(t, err)
		assert.Equal(t, "SCAM", flagType)
		assert.Equal(t, "CRITICAL", severity)
		assert.Equal(t, "automated_scanner", source)
		assert.Equal(t, "Contract has rug pull indicators", reason)
	})

	t.Run("can resolve quality flag", func(t *testing.T) {
		flagID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO quality_flags (id, asset_id, flag_type, severity, source, reason, raised_at)
			VALUES ($1, $2, 'SUSPICIOUS', 'WARNING', 'manual', 'Unusual trading pattern', $3)
		`, flagID, assetID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM quality_flags WHERE id = $1`, flagID)

		resolvedAt := now.Add(24 * time.Hour)
		_, err = db.Exec(`
			UPDATE quality_flags 
			SET resolved_at = $1, resolved_by = $2, resolution_notes = $3
			WHERE id = $4
		`, resolvedAt, "admin", "False alarm - trading pattern was legitimate", flagID)
		require.NoError(t, err)

		var resolved sql.NullTime
		var resolvedBy, resolutionNotes sql.NullString
		err = db.QueryRow(`SELECT resolved_at, resolved_by, resolution_notes FROM quality_flags WHERE id = $1`, flagID).
			Scan(&resolved, &resolvedBy, &resolutionNotes)
		require.NoError(t, err)
		assert.True(t, resolved.Valid)
		assert.True(t, resolvedBy.Valid)
		assert.Equal(t, "admin", resolvedBy.String)
		assert.True(t, resolutionNotes.Valid)
	})
}

func TestMigrations_AssetGroupsTable(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Setup: Create test assets
	ethID := uuid.New()
	wethID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	_, err := db.Exec(`
		INSERT INTO assets (id, symbol, name, type, created_at, updated_at)
		VALUES 
			($1, 'ETH', 'Ethereum', 'CRYPTOCURRENCY', $3, $3),
			($2, 'WETH', 'Wrapped ETH', 'WRAPPED', $3, $3)
	`, ethID, wethID, now)
	require.NoError(t, err)
	defer db.Exec(`DELETE FROM assets WHERE id IN ($1, $2)`, ethID, wethID)

	t.Run("tables exist", func(t *testing.T) {
		for _, tableName := range []string{"asset_groups", "group_members"} {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM information_schema.tables 
					WHERE table_schema = 'public' 
					AND table_name = $1
				)
			`, tableName).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "%s table should exist", tableName)
		}
	})

	t.Run("enforces group name format", func(t *testing.T) {
		groupID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO asset_groups (id, name, created_at, updated_at)
			VALUES ($1, 'Invalid Group Name', $2, $2)
		`, groupID, now)
		assert.Error(t, err, "Should fail with invalid group name format")
		assert.Contains(t, err.Error(), "chk_group_name_format", "Error should mention name format constraint")
	})

	t.Run("can create group and add members", func(t *testing.T) {
		groupID := uuid.New()
		memberID1 := uuid.New()
		memberID2 := uuid.New()

		_, err := db.Exec(`
			INSERT INTO asset_groups (id, name, description, created_at, updated_at)
			VALUES ($1, 'all_eth_variants', 'All ETH and ETH derivatives', $2, $2)
		`, groupID, now)
		require.NoError(t, err)
		defer func() {
			db.Exec(`DELETE FROM group_members WHERE group_id = $1`, groupID)
			db.Exec(`DELETE FROM asset_groups WHERE id = $1`, groupID)
		}()

		_, err = db.Exec(`
			INSERT INTO group_members (id, group_id, asset_id, weight, added_at)
			VALUES 
				($1, $2, $3, 1.0, $5),
				($4, $2, $6, 1.0, $5)
		`, memberID1, groupID, ethID, memberID2, now, wethID)
		require.NoError(t, err)

		var count int
		err = db.QueryRow(`SELECT COUNT(*) FROM group_members WHERE group_id = $1`, groupID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("enforces unique group+asset constraint", func(t *testing.T) {
		groupID := uuid.New()
		memberID1 := uuid.New()
		memberID2 := uuid.New()

		_, err := db.Exec(`
			INSERT INTO asset_groups (id, name, created_at, updated_at)
			VALUES ($1, 'test_group', $2, $2)
		`, groupID, now)
		require.NoError(t, err)
		defer func() {
			db.Exec(`DELETE FROM group_members WHERE group_id = $1`, groupID)
			db.Exec(`DELETE FROM asset_groups WHERE id = $1`, groupID)
		}()

		_, err = db.Exec(`
			INSERT INTO group_members (id, group_id, asset_id, added_at)
			VALUES ($1, $2, $3, $4)
		`, memberID1, groupID, ethID, now)
		require.NoError(t, err)

		// Try to add same asset to same group again
		_, err = db.Exec(`
			INSERT INTO group_members (id, group_id, asset_id, added_at)
			VALUES ($1, $2, $3, $4)
		`, memberID2, groupID, ethID, now)
		assert.Error(t, err, "Should fail with duplicate group+asset")
		assert.Contains(t, err.Error(), "unique_group_asset", "Error should mention unique constraint")
	})
}

func TestMigrations_AssetIdentifiersTable(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Setup: Create test asset
	assetID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	_, err := db.Exec(`
		INSERT INTO assets (id, symbol, name, type, created_at, updated_at)
		VALUES ($1, 'BTC', 'Bitcoin', 'CRYPTOCURRENCY', $2, $2)
	`, assetID, now)
	require.NoError(t, err)
	defer func() {
		db.Exec(`DELETE FROM asset_identifiers WHERE asset_id = $1`, assetID)
		db.Exec(`DELETE FROM assets WHERE id = $1`, assetID)
	}()

	t.Run("table exists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'asset_identifiers'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "asset_identifiers table should exist")
	})

	t.Run("has required indexes including GIN", func(t *testing.T) {
		expectedIndexes := []string{
			"idx_asset_identifiers_asset_id",
			"idx_asset_identifiers_source",
			"idx_asset_identifiers_external_id",
			"idx_asset_identifiers_source_external",
			"idx_asset_identifiers_metadata",
		}

		for _, indexName := range expectedIndexes {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM pg_indexes
					WHERE schemaname = 'public'
					AND tablename = 'asset_identifiers'
					AND indexname = $1
				)
			`, indexName).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Index %s should exist", indexName)
		}
	})

	t.Run("can insert and query asset identifier", func(t *testing.T) {
		identifierID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO asset_identifiers (id, asset_id, source, external_id, is_primary, created_at, updated_at)
			VALUES ($1, $2, 'coingecko', 'bitcoin', true, $3, $3)
		`, identifierID, assetID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM asset_identifiers WHERE id = $1`, identifierID)

		var source, externalID string
		var isPrimary bool
		err = db.QueryRow(`SELECT source, external_id, is_primary FROM asset_identifiers WHERE id = $1`, identifierID).
			Scan(&source, &externalID, &isPrimary)
		require.NoError(t, err)
		assert.Equal(t, "coingecko", source)
		assert.Equal(t, "bitcoin", externalID)
		assert.True(t, isPrimary)
	})

	t.Run("enforces unique asset+source constraint", func(t *testing.T) {
		identifierID1 := uuid.New()
		identifierID2 := uuid.New()

		_, err := db.Exec(`
			INSERT INTO asset_identifiers (id, asset_id, source, external_id, created_at, updated_at)
			VALUES ($1, $2, 'coingecko', 'bitcoin', $3, $3)
		`, identifierID1, assetID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM asset_identifiers WHERE id = $1`, identifierID1)

		_, err = db.Exec(`
			INSERT INTO asset_identifiers (id, asset_id, source, external_id, created_at, updated_at)
			VALUES ($1, $2, 'coingecko', 'bitcoin_different', $3, $3)
		`, identifierID2, assetID, now)
		assert.Error(t, err, "Should fail with duplicate asset+source")
		assert.Contains(t, err.Error(), "unique_asset_source", "Error should mention unique constraint")
	})
}

func TestMigrations_SymbolIdentifiersTable(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Setup: Create test assets and symbol
	baseAssetID := uuid.New()
	quoteAssetID := uuid.New()
	symbolID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	_, err := db.Exec(`
		INSERT INTO assets (id, symbol, name, type, created_at, updated_at)
		VALUES 
			($1, 'BTC', 'Bitcoin', 'CRYPTOCURRENCY', $3, $3),
			($2, 'USDT', 'Tether', 'STABLECOIN', $3, $3)
	`, baseAssetID, quoteAssetID, now)
	require.NoError(t, err)
	defer db.Exec(`DELETE FROM assets WHERE id IN ($1, $2)`, baseAssetID, quoteAssetID)

	_, err = db.Exec(`
		INSERT INTO symbols (id, base_asset_id, quote_asset_id, symbol_type, tick_size, lot_size, min_order_size, max_order_size, created_at, updated_at)
		VALUES ($1, $2, $3, 'SPOT', 0.01, 0.001, 0.001, 1000, $4, $4)
	`, symbolID, baseAssetID, quoteAssetID, now)
	require.NoError(t, err)
	defer func() {
		db.Exec(`DELETE FROM symbol_identifiers WHERE symbol_id = $1`, symbolID)
		db.Exec(`DELETE FROM symbols WHERE id = $1`, symbolID)
	}()

	t.Run("table exists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'symbol_identifiers'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "symbol_identifiers table should exist")
	})

	t.Run("can insert and query symbol identifier", func(t *testing.T) {
		identifierID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO symbol_identifiers (id, symbol_id, source, external_id, is_primary, created_at, updated_at)
			VALUES ($1, $2, 'coingecko', 'btc-usdt', true, $3, $3)
		`, identifierID, symbolID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM symbol_identifiers WHERE id = $1`, identifierID)

		var source, externalID string
		err = db.QueryRow(`SELECT source, external_id FROM symbol_identifiers WHERE id = $1`, identifierID).
			Scan(&source, &externalID)
		require.NoError(t, err)
		assert.Equal(t, "coingecko", source)
		assert.Equal(t, "btc-usdt", externalID)
	})
}

func TestMigrations_VenueAssetsTable(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Setup: Create test venue, asset
	venueID := "binance"
	assetID := uuid.New()
	chainID := "ethereum"
	now := time.Now().UTC().Truncate(time.Microsecond)

	_, err := db.Exec(`INSERT INTO chains (id, name, chain_type, created_at) VALUES ($1, 'Ethereum', 'EVM', $2)`, chainID, now)
	require.NoError(t, err)
	defer db.Exec(`DELETE FROM chains WHERE id = $1`, chainID)

	_, err = db.Exec(`
		INSERT INTO venues (id, name, venue_type, created_at)
		VALUES ($1, 'Binance', 'CEX', $2)
	`, venueID, now)
	require.NoError(t, err)
	defer db.Exec(`DELETE FROM venues WHERE id = $1`, venueID)

	_, err = db.Exec(`
		INSERT INTO assets (id, symbol, name, type, created_at, updated_at)
		VALUES ($1, 'BTC', 'Bitcoin', 'CRYPTOCURRENCY', $2, $2)
	`, assetID, now)
	require.NoError(t, err)
	defer func() {
		db.Exec(`DELETE FROM venue_assets WHERE venue_id = $1 OR asset_id = $2`, venueID, assetID)
		db.Exec(`DELETE FROM assets WHERE id = $1`, assetID)
	}()

	t.Run("table exists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'venue_assets'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "venue_assets table should exist")
	})

	t.Run("has required indexes", func(t *testing.T) {
		expectedIndexes := []string{
			"idx_venue_assets_venue_id",
			"idx_venue_assets_asset_id",
			"idx_venue_assets_venue_asset",
			"idx_venue_assets_trading_enabled",
		}

		for _, indexName := range expectedIndexes {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM pg_indexes
					WHERE schemaname = 'public'
					AND tablename = 'venue_assets'
					AND indexname = $1
				)
			`, indexName).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Index %s should exist", indexName)
		}
	})

	t.Run("can insert and query venue asset", func(t *testing.T) {
		venueAssetID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO venue_assets (id, venue_id, asset_id, venue_symbol, deposit_enabled, withdraw_enabled, trading_enabled, withdraw_fee, created_at, updated_at)
			VALUES ($1, $2, $3, 'BTC', true, true, true, 0.0005, $4, $4)
		`, venueAssetID, venueID, assetID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM venue_assets WHERE id = $1`, venueAssetID)

		var venueSymbol string
		var depositEnabled, withdrawEnabled, tradingEnabled bool
		err = db.QueryRow(`SELECT venue_symbol, deposit_enabled, withdraw_enabled, trading_enabled FROM venue_assets WHERE id = $1`, venueAssetID).
			Scan(&venueSymbol, &depositEnabled, &withdrawEnabled, &tradingEnabled)
		require.NoError(t, err)
		assert.Equal(t, "BTC", venueSymbol)
		assert.True(t, depositEnabled)
		assert.True(t, withdrawEnabled)
		assert.True(t, tradingEnabled)
	})

	t.Run("enforces unique venue+asset constraint", func(t *testing.T) {
		venueAssetID1 := uuid.New()
		venueAssetID2 := uuid.New()

		_, err := db.Exec(`
			INSERT INTO venue_assets (id, venue_id, asset_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $4)
		`, venueAssetID1, venueID, assetID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM venue_assets WHERE id = $1`, venueAssetID1)

		_, err = db.Exec(`
			INSERT INTO venue_assets (id, venue_id, asset_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $4)
		`, venueAssetID2, venueID, assetID, now)
		assert.Error(t, err, "Should fail with duplicate venue+asset")
		assert.Contains(t, err.Error(), "unique_venue_asset", "Error should mention unique constraint")
	})
}

func TestMigrations_VenueSymbolsTable(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Setup: Create test venue, assets, symbol
	venueID := "binance"
	baseAssetID := uuid.New()
	quoteAssetID := uuid.New()
	symbolID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	_, err := db.Exec(`
		INSERT INTO venues (id, name, venue_type, created_at)
		VALUES ($1, 'Binance', 'CEX', $2)
	`, venueID, now)
	require.NoError(t, err)
	defer db.Exec(`DELETE FROM venues WHERE id = $1`, venueID)

	_, err = db.Exec(`
		INSERT INTO assets (id, symbol, name, type, created_at, updated_at)
		VALUES 
			($1, 'BTC', 'Bitcoin', 'CRYPTOCURRENCY', $3, $3),
			($2, 'USDT', 'Tether', 'STABLECOIN', $3, $3)
	`, baseAssetID, quoteAssetID, now)
	require.NoError(t, err)
	defer db.Exec(`DELETE FROM assets WHERE id IN ($1, $2)`, baseAssetID, quoteAssetID)

	_, err = db.Exec(`
		INSERT INTO symbols (id, base_asset_id, quote_asset_id, symbol_type, tick_size, lot_size, min_order_size, max_order_size, created_at, updated_at)
		VALUES ($1, $2, $3, 'SPOT', 0.01, 0.001, 0.001, 1000, $4, $4)
	`, symbolID, baseAssetID, quoteAssetID, now)
	require.NoError(t, err)
	defer func() {
		db.Exec(`DELETE FROM venue_symbols WHERE venue_id = $1 OR symbol_id = $2`, venueID, symbolID)
		db.Exec(`DELETE FROM symbols WHERE id = $1`, symbolID)
	}()

	t.Run("table exists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'venue_symbols'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "venue_symbols table should exist")
	})

	t.Run("has required indexes including critical cqmd lookup", func(t *testing.T) {
		expectedIndexes := []string{
			"idx_venue_symbols_venue_id",
			"idx_venue_symbols_symbol_id",
			"idx_venue_symbols_venue_string", // CRITICAL for GetVenueSymbol(venue_id, venue_symbol) queries
		}

		for _, indexName := range expectedIndexes {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM pg_indexes
					WHERE schemaname = 'public'
					AND tablename = 'venue_symbols'
					AND indexname = $1
				)
			`, indexName).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Index %s should exist", indexName)
		}
	})

	t.Run("can insert and query venue symbol", func(t *testing.T) {
		venueSymbolID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO venue_symbols (id, venue_id, symbol_id, venue_symbol, maker_fee, taker_fee, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, 'BTCUSDT', 0.001, 0.001, true, $4, $4)
		`, venueSymbolID, venueID, symbolID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM venue_symbols WHERE id = $1`, venueSymbolID)

		var venueSymbol string
		var makerFee, takerFee float64
		var isActive bool
		err = db.QueryRow(`SELECT venue_symbol, maker_fee, taker_fee, is_active FROM venue_symbols WHERE id = $1`, venueSymbolID).
			Scan(&venueSymbol, &makerFee, &takerFee, &isActive)
		require.NoError(t, err)
		assert.Equal(t, "BTCUSDT", venueSymbol)
		assert.Equal(t, 0.001, makerFee)
		assert.Equal(t, 0.001, takerFee)
		assert.True(t, isActive)
	})

	t.Run("enforces unique venue+symbol constraint", func(t *testing.T) {
		venueSymbolID1 := uuid.New()
		venueSymbolID2 := uuid.New()

		_, err := db.Exec(`
			INSERT INTO venue_symbols (id, venue_id, symbol_id, venue_symbol, created_at, updated_at)
			VALUES ($1, $2, $3, 'BTCUSDT', $4, $4)
		`, venueSymbolID1, venueID, symbolID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM venue_symbols WHERE id = $1`, venueSymbolID1)

		_, err = db.Exec(`
			INSERT INTO venue_symbols (id, venue_id, symbol_id, venue_symbol, created_at, updated_at)
			VALUES ($1, $2, $3, 'BTCUSDT2', $4, $4)
		`, venueSymbolID2, venueID, symbolID, now)
		assert.Error(t, err, "Should fail with duplicate venue+symbol")
		assert.Contains(t, err.Error(), "unique_venue_symbol", "Error should mention unique constraint")
	})

	t.Run("enforces unique venue_symbol string per venue", func(t *testing.T) {
		symbolID2 := uuid.New()
		venueSymbolID1 := uuid.New()
		venueSymbolID2 := uuid.New()

		// Create another symbol
		_, err := db.Exec(`
			INSERT INTO symbols (id, base_asset_id, quote_asset_id, symbol_type, tick_size, lot_size, min_order_size, max_order_size, created_at, updated_at)
			VALUES ($1, $2, $3, 'SPOT', 0.01, 0.001, 0.001, 1000, $4, $4)
		`, symbolID2, baseAssetID, quoteAssetID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM symbols WHERE id = $1`, symbolID2)

		_, err = db.Exec(`
			INSERT INTO venue_symbols (id, venue_id, symbol_id, venue_symbol, created_at, updated_at)
			VALUES ($1, $2, $3, 'BTCUSDT', $4, $4)
		`, venueSymbolID1, venueID, symbolID, now)
		require.NoError(t, err)
		defer db.Exec(`DELETE FROM venue_symbols WHERE id = $1`, venueSymbolID1)

		// Try to insert different symbol with same venue_symbol string
		_, err = db.Exec(`
			INSERT INTO venue_symbols (id, venue_id, symbol_id, venue_symbol, created_at, updated_at)
			VALUES ($1, $2, $3, 'BTCUSDT', $4, $4)
		`, venueSymbolID2, venueID, symbolID2, now)
		assert.Error(t, err, "Should fail with duplicate venue_symbol string")
		assert.Contains(t, err.Error(), "unique_venue_symbol_string", "Error should mention unique venue_symbol_string constraint")
	})

	t.Run("enforces fee range constraints", func(t *testing.T) {
		venueSymbolID := uuid.New()

		_, err := db.Exec(`
			INSERT INTO venue_symbols (id, venue_id, symbol_id, venue_symbol, maker_fee, taker_fee, created_at, updated_at)
			VALUES ($1, $2, $3, 'TESTBTC', 1.5, 0.001, $4, $4)
		`, venueSymbolID, venueID, symbolID, now)
		assert.Error(t, err, "Should fail with maker_fee > 100%")
		assert.Contains(t, err.Error(), "chk_maker_fee_range", "Error should mention maker_fee constraint")
	})
}
