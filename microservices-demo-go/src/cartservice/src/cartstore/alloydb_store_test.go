package cartstore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlloyDBCartStore(t *testing.T) {
	// Connection string for the docker postgres container
	// user=postgres password=mysecretpassword dbname=postgres host=localhost port=5432
	connString := "postgres://postgres:mysecretpassword@localhost:5432/postgres"
	tableName := "cartitems"

	ctx := context.Background()

	// Wait for DB to be ready
	var pool *pgxpool.Pool
	var err error
	for i := 0; i < 10; i++ {
		pool, err = pgxpool.New(ctx, connString)
		if err == nil {
			err = pool.Ping(ctx)
			if err == nil {
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
	require.NoError(t, err, "Failed to connect to Postgres")
	defer pool.Close()

	// Setup Schema
	_, err = pool.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			userId text NOT NULL,
			productId text NOT NULL,
			quantity integer NOT NULL,
			PRIMARY KEY (userId, productId)
		)
	`, tableName))
	require.NoError(t, err, "Failed to create table")

	// Cleanup before test
	_, err = pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s", tableName))
	require.NoError(t, err)

	// Create Store
	store, err := NewAlloyDBCartStore(ctx, connString, tableName)
	require.NoError(t, err)
	defer store.Close()

	userID := "test-user-alloy"

	t.Run("AddItem", func(t *testing.T) {
		err := store.AddItem(ctx, userID, "item-1", 1)
		require.NoError(t, err)

		cart, err := store.GetCart(ctx, userID)
		require.NoError(t, err)
		require.Len(t, cart.Items, 1)
		assert.Equal(t, "item-1", cart.Items[0].ProductId)
		assert.Equal(t, int32(1), cart.Items[0].Quantity)
	})

	t.Run("AddItem increment", func(t *testing.T) {
		err := store.AddItem(ctx, userID, "item-1", 2)
		require.NoError(t, err)

		cart, err := store.GetCart(ctx, userID)
		require.NoError(t, err)
		require.Len(t, cart.Items, 1)
		assert.Equal(t, int32(3), cart.Items[0].Quantity)
	})

	t.Run("EmptyCart", func(t *testing.T) {
		err := store.EmptyCart(ctx, userID)
		require.NoError(t, err)

		cart, err := store.GetCart(ctx, userID)
		require.NoError(t, err)
		assert.Nil(t, cart.Items)
	})
}
