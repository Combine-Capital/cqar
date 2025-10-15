package repository

import (
	"context"

	"github.com/Combine-Capital/cqi/pkg/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PostgresRepository implements the Repository interface using PostgreSQL via CQI
type PostgresRepository struct {
	pool *database.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository instance
func NewPostgresRepository(pool *database.Pool) Repository {
	return &PostgresRepository{
		pool: pool,
	}
}

// WithTransaction executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (r *PostgresRepository) WithTransaction(ctx context.Context, fn func(repo Repository) error) error {
	return r.pool.WithTransaction(ctx, func(tx database.Transaction) error {
		// Create a transaction-scoped repository
		txRepo := &postgresTxRepository{
			tx:                 tx,
			PostgresRepository: r,
		}
		return fn(txRepo)
	})
}

// Ping checks the database connection
func (r *PostgresRepository) Ping(ctx context.Context) error {
	return r.pool.HealthCheck(ctx)
}

// exec is a helper to execute a query and return the result
func (r *PostgresRepository) exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return r.pool.Exec(ctx, query, args...)
}

// queryRow is a helper to query a single row
func (r *PostgresRepository) queryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return r.pool.QueryRow(ctx, query, args...)
}

// query is a helper to query multiple rows
func (r *PostgresRepository) query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return r.pool.Query(ctx, query, args...)
}

// postgresTxRepository wraps a transaction to provide repository methods within a transaction
type postgresTxRepository struct {
	tx database.Transaction
	*PostgresRepository
}

// Override exec/queryRow/query to use the transaction instead of the pool
func (r *postgresTxRepository) exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return r.tx.Exec(ctx, query, args...)
}

func (r *postgresTxRepository) queryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return r.tx.QueryRow(ctx, query, args...)
}

func (r *postgresTxRepository) query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return r.tx.Query(ctx, query, args...)
}

// ptr helper functions for working with protobuf pointer fields
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func int32Ptr(i int32) *int32 {
	if i == 0 {
		return nil
	}
	return &i
}

func int32Val(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

func float64Ptr(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}

func float64Val(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func boolPtr(b bool) *bool {
	return &b
}

func boolVal(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
