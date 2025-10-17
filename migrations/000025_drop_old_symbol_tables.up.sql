-- Drop old symbol-related tables that are replaced by instruments/markets
DROP TABLE IF EXISTS venue_symbols CASCADE;
DROP TABLE IF EXISTS symbol_identifiers CASCADE;
DROP TABLE IF EXISTS symbols CASCADE;

-- Drop old asset_identifiers table (replaced by unified identifiers table)
DROP TABLE IF EXISTS asset_identifiers CASCADE;
