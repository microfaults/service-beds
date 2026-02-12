package db

import "context"

type Product struct {
	ID          string
	Description string
}

type ProductStore interface {
	SearchProducts(ctx context.Context, embedding []float32) ([]Product, error)
}
