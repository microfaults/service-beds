package cartstore

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/model"
)

type MemoryCartStore struct {
	mu    sync.RWMutex
	carts map[string][]byte // Storing as JSON bytes to simulate serialization behavior
}

func NewMemoryCartStore() *MemoryCartStore {
	return &MemoryCartStore{
		carts: make(map[string][]byte),
	}
}

func (s *MemoryCartStore) AddItem(ctx context.Context, userID, productID string, quantity int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get existing cart (logic duplicated from Redis implementation for simulation)
	var cart *model.Cart
	if data, ok := s.carts[userID]; ok {
		cart = &model.Cart{}
		if err := json.Unmarshal(data, cart); err != nil {
			return fmt.Errorf("failed to unmarshal cart: %w", err)
		}
	} else {
		cart = &model.Cart{UserID: userID, Items: []*model.CartItem{}}
	}

	cart.AddItem(productID, quantity)

	data, err := json.Marshal(cart)
	if err != nil {
		return fmt.Errorf("failed to marshal cart: %w", err)
	}
	s.carts[userID] = data
	return nil
}

func (s *MemoryCartStore) GetCart(ctx context.Context, userID string) (*model.Cart, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if data, ok := s.carts[userID]; ok {
		cart := &model.Cart{}
		if err := json.Unmarshal(data, cart); err != nil {
			return nil, fmt.Errorf("failed to unmarshal cart: %w", err)
		}
		return cart, nil
	}

	return &model.Cart{UserID: userID, Items: []*model.CartItem{}}, nil
}

func (s *MemoryCartStore) EmptyCart(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.carts, userID)
	return nil
}
