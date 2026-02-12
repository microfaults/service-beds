package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type PostgresStore struct {
	pool      *pgxpool.Pool
	tableName string
}

// NewPostgresStore creates a new PostgresStore using a connection string (DSN).
// It handles the connection pool creation.
func NewPostgresStore(ctx context.Context, dsn, tableName string) (*PostgresStore, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dsn: %w", err)
	}

	// Set some default pool settings
	config.MaxConns = 10
	config.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStore{
		pool:      pool,
		tableName: tableName,
	}, nil
}

func (s *PostgresStore) SearchProducts(ctx context.Context, embeddingValues []float32) ([]Product, error) {
	embedding := pgvector.NewVector(embeddingValues)

	rows, err := s.pool.Query(ctx,
		fmt.Sprintf("SELECT id, description FROM %s ORDER BY product_embedding <=> $1 LIMIT 5", s.tableName),
		embedding,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var id, desc string
		if err := rows.Scan(&id, &desc); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		products = append(products, Product{ID: id, Description: desc})
	}

	return products, nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}
