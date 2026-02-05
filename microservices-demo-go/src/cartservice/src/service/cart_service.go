package service

import (
	"context"

	"github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/cartstore"
	pb "github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/proto"
)

type CartService struct {
	pb.UnimplementedCartServiceServer
	store cartstore.CartStore
}

func NewCartService(store cartstore.CartStore) *CartService {
	return &CartService{store: store}
}

func (s *CartService) AddItem(ctx context.Context, req *pb.AddItemRequest) (*pb.Empty, error) {
	err := s.store.AddItem(ctx, req.UserId, req.Item.ProductId, req.Item.Quantity)
	if err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

func (s *CartService) GetCart(ctx context.Context, req *pb.GetCartRequest) (*pb.Cart, error) {
	return s.store.GetCart(ctx, req.UserId)
}

func (s *CartService) EmptyCart(ctx context.Context, req *pb.EmptyCartRequest) (*pb.Empty, error) {
	err := s.store.EmptyCart(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}
