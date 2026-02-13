package cartstore

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlloyDBCartStore struct {
	pool      *pgxpool.Pool
	tableName string
}

func NewAlloyDBCartStore(ctx context.Context, pool *pgxpool.Pool, tableName string) *AlloyDBCartStore {
	return &AlloyDBCartStore{
		pool:      pool,
		tableName: tableName,
	}
}

func (s *AlloyDBCartStore) AddItem(ctx context.Context, userID, productID string, quantity int32) error {
	// Transaction to safely update quantity
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Get current quantity
	var currentQuantity int32
	query := fmt.Sprintf("SELECT quantity FROM %s WHERE userId = $1 AND productId = $2", s.tableName)
	err = tx.QueryRow(ctx, query, userID, productID).Scan(&currentQuantity)
	if err != nil {
		// If not found, quantity is 0, which is fine
		currentQuantity = 0
	}

	newQuantity := currentQuantity + quantity

	// 2. Upsert
	// PostgreSQL/AlloyDB INSERT ... ON CONFLICT
	upsertQuery := fmt.Sprintf(`
        INSERT INTO %s (userId, productId, quantity)
        VALUES ($1, $2, $3)
        ON CONFLICT (userId, productId)
        DO UPDATE SET quantity = $3
    `, s.tableName)

	_, err = tx.Exec(ctx, upsertQuery, userID, productID, newQuantity)
	if err != nil {
		return fmt.Errorf("failed to upsert item: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *AlloyDBCartStore) GetCart(ctx context.Context, userID string) (*model.Cart, error) {
	query := fmt.Sprintf("SELECT productId, quantity FROM %s WHERE userId = $1", s.tableName)
	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cart: %w", err)
	}
	defer rows.Close()

	cart := &model.Cart{
		UserID: userID,
		Items:  make([]*model.CartItem, 0),
	}

	for rows.Next() {
		var item model.CartItem
		if err := rows.Scan(&item.ProductID, &item.Quantity); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		cart.Items = append(cart.Items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return cart, nil
}

func (s *AlloyDBCartStore) EmptyCart(ctx context.Context, userID string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE userId = $1", s.tableName)
	_, err := s.pool.Exec(ctx, query, userID)
	return err
}

// ConnectAlloyDB creates a connection pool to AlloyDB/Postgres
func ConnectAlloyDB(ctx context.Context, ip, user, password, dbname string) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s", user, password, ip, dbname)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Connection pool settings matching typical microservice needs
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
