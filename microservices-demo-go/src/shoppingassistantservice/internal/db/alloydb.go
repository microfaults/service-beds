package db

import (
	"context"
	"fmt"
	"net"

	"cloud.google.com/go/alloydbconn"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type AlloyDBStore struct {
	pool      *pgxpool.Pool
	dialer    *alloydbconn.Dialer
	tableName string
}

// NewAlloyDBStore creates a new AlloyDBStore.
// It handles fetching the password from Secret Manager and setting up the AlloyDB Dialer.
func NewAlloyDBStore(ctx context.Context, projectID, region, cluster, instance, dbName, secretName, tableName string) (*AlloyDBStore, error) {
	// 1. Fetch DB Password from Secret Manager
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close()

	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName),
	}

	result, err := client.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to access secret: %w", err)
	}
	dbPassword := string(result.Payload.Data)

	// 2. Initialize AlloyDB Connection
	dialer, err := alloydbconn.NewDialer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create alloydb dialer: %w", err)
	}

	dsn := fmt.Sprintf("user=postgres password=%s dbname=%s sslmode=disable", dbPassword, dbName)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		dialer.Close()
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config.ConnConfig.DialFunc = func(ctx context.Context, _ string, _ string) (net.Conn, error) {
		return dialer.Dial(ctx, fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", projectID, region, cluster, instance))
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		dialer.Close()
		return nil, fmt.Errorf("failed to connect to alloydb: %w", err)
	}

	return &AlloyDBStore{
		pool:      pool,
		dialer:    dialer,
		tableName: tableName,
	}, nil
}

func (s *AlloyDBStore) SearchProducts(ctx context.Context, embeddingValues []float32) ([]Product, error) {
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

func (s *AlloyDBStore) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
	if s.dialer != nil {
		s.dialer.Close()
	}
}

// We can just use one struct `PGVectorStore`.
