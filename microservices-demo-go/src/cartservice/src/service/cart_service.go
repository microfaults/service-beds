package service

import (
	"context"

	"github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/cartstore"
	"github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/model"
)

type CartService struct {
	store cartstore.CartStore
}

func NewCartService(store cartstore.CartStore) *CartService {
	return &CartService{store: store}
}

func (s *CartService) AddItem(ctx context.Context, userID, productID string, quantity int32) error {
	return s.store.AddItem(ctx, userID, productID, quantity)
}

func (s *CartService) GetCart(ctx context.Context, userID string) (*model.Cart, error) {
	return s.store.GetCart(ctx, userID)
}

func (s *CartService) EmptyCart(ctx context.Context, userID string) error {
	return s.store.EmptyCart(ctx, userID)
}
