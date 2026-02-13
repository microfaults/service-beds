package cartstore

import (
	"context"

	"github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/model"
)

type CartStore interface {
	AddItem(ctx context.Context, userID, productID string, quantity int32) error
	GetCart(ctx context.Context, userID string) (*model.Cart, error)
	EmptyCart(ctx context.Context, userID string) error
}
