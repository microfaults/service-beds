package cartstore

import (
	"context"
	"fmt"
	"sync"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/proto"
	"google.golang.org/protobuf/proto"
)

type MemoryCartStore struct {
	mu    sync.RWMutex
	carts map[string][]byte
}

func NewMemoryCartStore() *MemoryCartStore {
	return &MemoryCartStore{
		carts: make(map[string][]byte),
	}
}

func (s *MemoryCartStore) AddItem(ctx context.Context, userID, productID string, quantity int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get existing cart
	var cart *pb.Cart
	if data, ok := s.carts[userID]; ok {
		cart = &pb.Cart{}
		if err := proto.Unmarshal(data, cart); err != nil {
			return fmt.Errorf("failed to unmarshal cart: %w", err)
		}
	} else {
		cart = &pb.Cart{UserId: userID}
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

	// Save back to memory
	data, err := proto.Marshal(cart)
	if err != nil {
		return fmt.Errorf("failed to marshal cart: %w", err)
	}
	s.carts[userID] = data
	return nil
}

func (s *MemoryCartStore) GetCart(ctx context.Context, userID string) (*pb.Cart, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if data, ok := s.carts[userID]; ok {
		cart := &pb.Cart{}
		if err := proto.Unmarshal(data, cart); err != nil {
			return nil, fmt.Errorf("failed to unmarshal cart: %w", err)
		}
		return cart, nil
	}

	return &pb.Cart{UserId: userID}, nil
}

func (s *MemoryCartStore) EmptyCart(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.carts, userID)
	return nil
}
