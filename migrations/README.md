# Database Migrations

This directory contains database migrations for the CQAR service using [golang-migrate](https://github.com/golang-migrate/migrate).

## Migration Files

Migrations are numbered sequentially and come in pairs:

- `NNNNNN_description.up.sql` - Forward migration
- `NNNNNN_description.down.sql` - Rollback migration

### Current Migrations

1. **000001_create_assets_table** - Core assets table with canonical UUIDs
2. **000002_create_symbols_table** - Trading pairs/markets with base/quote assets
3. **000003_create_chains_table** - Blockchain networks registry
4. **000004_create_venues_table** - Trading venues (CEX/DEX/DeFi protocols)

## Prerequisites

### Install golang-migrate

**macOS:**
```bash
brew install golang-migrate
```

**Linux:**
```bash
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/
```

**Go install:**
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### Start Development Database

Using docker-compose:
```bash
docker-compose up -d postgres
```

Wait for PostgreSQL to be ready:
```bash
docker-compose logs -f postgres
# Wait for "database system is ready to accept connections"
```

## Running Migrations

### Apply All Migrations

```bash
make migrate-up
```

Or manually:
```bash
migrate -path ./migrations -database "postgres://cqar:cqar_dev_password@localhost:5432/cqar?sslmode=disable" up
```

### Rollback Last Migration

```bash
make migrate-down
```

Or manually:
```bash
migrate -path ./migrations -database "postgres://cqar:cqar_dev_password@localhost:5432/cqar?sslmode=disable" down 1
```

### Check Migration Status

```bash
make migrate-status
```

Or manually:
```bash
migrate -path ./migrations -database "postgres://cqar:cqar_dev_password@localhost:5432/cqar?sslmode=disable" version
```

### Force Migration Version (Use with Caution!)

If migrations are in a dirty state:
```bash
make migrate-force VERSION=4
```

## Verification

### Verify Tables Created

```bash
docker exec -it cqar-postgres psql -U cqar -d cqar -c "\dt"
```

Expected output:
```
              List of relations
 Schema |     Name      | Type  | Owner 
--------+---------------+-------+-------
 public | assets        | table | cqar
 public | chains        | table | cqar
 public | symbols       | table | cqar
 public | venues        | table | cqar
```

### Verify Assets Table Schema

```bash
docker exec -it cqar-postgres psql -U cqar -d cqar -c "\d assets"
```

### Verify Indexes

```bash
docker exec -it cqar-postgres psql -U cqar -d cqar -c "\di"
```

Expected indexes:
- idx_assets_symbol
- idx_assets_type
- idx_assets_created_at
- idx_symbols_base_asset
- idx_symbols_quote_asset
- idx_symbols_type
- idx_symbols_base_quote
- idx_symbols_expiry
- idx_symbols_created_at
- idx_chains_type
- idx_chains_native_asset
- idx_venues_type
- idx_venues_chain_id
- idx_venues_active

### Verify Foreign Key Constraints

```bash
docker exec -it cqar-postgres psql -U cqar -d cqar -c "
SELECT
    tc.constraint_name,
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema = kcu.table_schema
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
    AND ccu.table_schema = tc.table_schema
WHERE tc.constraint_type = 'FOREIGN KEY';
"
```

Expected foreign keys:
- symbols.base_asset_id → assets.id
- symbols.quote_asset_id → assets.id
- symbols.settlement_asset_id → assets.id
- chains.native_asset_id → assets.id
- venues.chain_id → chains.id

## Test Data Insertion

### Insert Test Chain

```bash
docker exec -it cqar-postgres psql -U cqar -d cqar -c "
INSERT INTO chains (id, name, chain_type, rpc_urls, explorer_url)
VALUES ('ethereum', 'Ethereum Mainnet', 'EVM', 
        ARRAY['https://eth.llamarpc.com'], 'https://etherscan.io');
"
```

### Insert Test Asset

```bash
docker exec -it cqar-postgres psql -U cqar -d cqar -c "
INSERT INTO assets (id, symbol, name, type, category)
VALUES ('550e8400-e29b-41d4-a716-446655440000', 'ETH', 'Ethereum', 'CRYPTOCURRENCY', 'Layer1');
"
```

### Insert Test Symbol

```bash
docker exec -it cqar-postgres psql -U cqar -d cqar -c "
-- First insert USDT
INSERT INTO assets (id, symbol, name, type, category)
VALUES ('550e8400-e29b-41d4-a716-446655440001', 'USDT', 'Tether', 'STABLECOIN', 'Stablecoin');

-- Then create ETH/USDT spot symbol
INSERT INTO symbols (id, base_asset_id, quote_asset_id, symbol_type, 
                     tick_size, lot_size, min_order_size, max_order_size)
VALUES ('650e8400-e29b-41d4-a716-446655440000',
        '550e8400-e29b-41d4-a716-446655440000',
        '550e8400-e29b-41d4-a716-446655440001',
        'SPOT', 0.01, 0.001, 0.001, 1000.0);
"
```

### Verify Foreign Key Enforcement

This should fail (invalid asset_id):
```bash
docker exec -it cqar-postgres psql -U cqar -d cqar -c "
INSERT INTO symbols (id, base_asset_id, quote_asset_id, symbol_type, 
                     tick_size, lot_size, min_order_size, max_order_size)
VALUES ('750e8400-e29b-41d4-a716-446655440000',
        '00000000-0000-0000-0000-000000000000',
        '550e8400-e29b-41d4-a716-446655440001',
        'SPOT', 0.01, 0.001, 0.001, 1000.0);
"
```

Expected error: `ERROR:  insert or update on table "symbols" violates foreign key constraint`

## Creating New Migrations

```bash
make migrate-create NAME=create_new_table
```

This will create:
- `migrations/YYYYMMDDHHMMSS_create_new_table.up.sql`
- `migrations/YYYYMMDDHHMMSS_create_new_table.down.sql`

## Troubleshooting

### Dirty Migration State

If a migration fails partway through, the migration state may be "dirty":

```bash
migrate -path ./migrations -database "postgres://..." version
# Output: 3/d
```

To fix:
1. Manually clean up any partial changes in the database
2. Force the version to the last successful migration:
   ```bash
   make migrate-force VERSION=2
   ```
3. Try running migrations again

### Connection Issues

Verify PostgreSQL is running and accessible:
```bash
docker-compose ps postgres
docker-compose logs postgres
```

Test connection:
```bash
docker exec -it cqar-postgres psql -U cqar -d cqar -c "SELECT 1;"
```

### Schema Conflicts

If you need to completely reset the database:
```bash
docker-compose down -v  # Removes volumes
docker-compose up -d postgres
make migrate-up
```

## Production Considerations

1. **Always test migrations in a staging environment first**
2. **Back up the database before running migrations in production**
3. **Review migration SQL before applying**
4. **Never force migration versions in production without investigation**
5. **Use transactions where possible** (golang-migrate wraps each migration in a transaction by default)
6. **Monitor migration execution time** for large tables
7. **Consider using `--lock-timeout` for production databases** to avoid blocking

## References

- [golang-migrate documentation](https://github.com/golang-migrate/migrate)
- [PostgreSQL documentation](https://www.postgresql.org/docs/)
- [CQAR SPEC.md](../docs/SPEC.md) - Database schema specification
