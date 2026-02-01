package cartstore

import (
	"context"
	"fmt"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/proto"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
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

	// Check if item exists
	found := false
	for _, item := range cart.Items {
		if item.ProductId == productID {
			item.Quantity += quantity
			found = true
			break
		}
	}

	// If not found, add new item
	if !found {
		cart.Items = append(cart.Items, &pb.CartItem{
			ProductId: productID,
			Quantity:  quantity,
		})
	}

	// Save back to Redis
	data, err := proto.Marshal(cart)
	if err != nil {
		return fmt.Errorf("failed to marshal cart: %w", err)
	}

	return s.client.Set(ctx, userID, data, 0).Err()
}

func (s *RedisCartStore) GetCart(ctx context.Context, userID string) (*pb.Cart, error) {
	val, err := s.client.Get(ctx, userID).Bytes()
	if err == redis.Nil {
		return &pb.Cart{UserId: userID}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get cart from redis: %w", err)
	}

	cart := &pb.Cart{}
	if err := proto.Unmarshal(val, cart); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cart: %w", err)
	}
	return cart, nil
}

func (s *RedisCartStore) EmptyCart(ctx context.Context, userID string) error {
	return s.client.Del(ctx, userID).Err()
}
