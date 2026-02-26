package cartstore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/model"
	"github.com/redis/go-redis/v9"
)

type RedisCartStore struct {
	client *redis.Client
}

func NewRedisCartStore(addr string) *RedisCartStore {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisCartStore{client: rdb}
}

func (s *RedisCartStore) AddItem(ctx context.Context, userID, productID string, quantity int32) error {
	// Get existing cart
	cart, err := s.GetCart(ctx, userID)
	if err != nil {
		return err
	}

	// Add item using helper
	cart.AddItem(productID, quantity)

	// Save back to Redis using JSON
	data, err := json.Marshal(cart)
	if err != nil {
		return fmt.Errorf("failed to marshal cart: %w", err)
	}

	return s.client.Set(ctx, userID, data, 0).Err()
}

func (s *RedisCartStore) GetCart(ctx context.Context, userID string) (*model.Cart, error) {
	val, err := s.client.Get(ctx, userID).Bytes()
	if err == redis.Nil {
		return &model.Cart{UserID: userID, Items: []*model.CartItem{}}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get cart from redis: %w", err)
	}

	cart := &model.Cart{}
	if err := json.Unmarshal(val, cart); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cart: %w", err)
	}
	return cart, nil
}

func (s *RedisCartStore) EmptyCart(ctx context.Context, userID string) error {
	return s.client.Del(ctx, userID).Err()
}
