package cartstore

import (
	"context"
	"fmt"
	"time"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/proto"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlloyDBCartStore struct {
	pool      *pgxpool.Pool
	tableName string
}

func NewAlloyDBCartStore(ctx context.Context, connectionString, tableName string) (*AlloyDBCartStore, error) {
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Basic configuration to match general best practices
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &AlloyDBCartStore{
		pool:      pool,
		tableName: tableName,
	}, nil
}

func (s *AlloyDBCartStore) AddItem(ctx context.Context, userID, productID string, quantity int32) error {
	// Logic mirrors C# implementation:
	// 1. Get current quantity
	// 2. Calculate new quantity
	// 3. Upsert (INSERT ... ON CONFLICT)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Get current quantity
	var currentQuantity int32
	// Use slightly safer SQL construction than the C# string interpolation
	query := fmt.Sprintf("SELECT quantity FROM %s WHERE userId = $1 AND productId = $2", s.tableName)
	err = tx.QueryRow(ctx, query, userID, productID).Scan(&currentQuantity)
	if err != nil {
		// If no row found, quantity is 0, which is fine
		currentQuantity = 0
	}

	newQuantity := currentQuantity + quantity

	// 3. Upsert
	upsert := fmt.Sprintf(`
		INSERT INTO %s (userId, productId, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (userId, productId)
		DO UPDATE SET quantity = $3
	`, s.tableName)

	_, err = tx.Exec(ctx, upsert, userID, productID, newQuantity)
	if err != nil {
		return fmt.Errorf("failed to upsert cart item: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *AlloyDBCartStore) GetCart(ctx context.Context, userID string) (*pb.Cart, error) {
	cart := &pb.Cart{UserId: userID}

	query := fmt.Sprintf("SELECT productId, quantity FROM %s WHERE userId = $1", s.tableName)
	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cart: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var productID string
		var quantity int32
		if err := rows.Scan(&productID, &quantity); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		cart.Items = append(cart.Items, &pb.CartItem{
			ProductId: productID,
			Quantity:  quantity,
		})
	}

	return cart, nil
}

func (s *AlloyDBCartStore) EmptyCart(ctx context.Context, userID string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE userId = $1", s.tableName)
	_, err := s.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to empty cart: %w", err)
	}
	return nil
}

func (s *AlloyDBCartStore) Close() {
	s.pool.Close()
}
