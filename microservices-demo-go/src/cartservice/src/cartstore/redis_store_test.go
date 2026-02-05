package cartstore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisCartStore(t *testing.T) {
	// Assumes Redis is running on localhost:6379
	store := NewRedisCartStore("127.0.0.1:6379")
	ctx := context.Background()
	userID := "test-user"

	// Ensure clean state
	err := store.EmptyCart(ctx, userID)
	require.NoError(t, err)

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
		require.Len(t, cart.Items, 1)                     // Should still be 1 item
		assert.Equal(t, int32(3), cart.Items[0].Quantity) // 1 + 2 = 3
	})

	t.Run("Add different Item", func(t *testing.T) {
		err := store.AddItem(ctx, userID, "item-2", 5)
		require.NoError(t, err)

		cart, err := store.GetCart(ctx, userID)
		require.NoError(t, err)
		require.Len(t, cart.Items, 2)

		// Map for easier verification
		items := make(map[string]int32)
		for _, item := range cart.Items {
			items[item.ProductId] = item.Quantity
		}
		assert.Equal(t, int32(3), items["item-1"])
		assert.Equal(t, int32(5), items["item-2"])
	})

	t.Run("EmptyCart", func(t *testing.T) {
		err := store.EmptyCart(ctx, userID)
		require.NoError(t, err)

		cart, err := store.GetCart(ctx, userID)
		require.NoError(t, err)
		assert.Nil(t, cart.Items) // Should be empty
	})
}
