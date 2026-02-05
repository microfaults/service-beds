package cartstore

import (
	"context"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/proto"
)

type CartStore interface {
	AddItem(ctx context.Context, userID, productID string, quantity int32) error
	GetCart(ctx context.Context, userID string) (*pb.Cart, error)
	EmptyCart(ctx context.Context, userID string) error
}
