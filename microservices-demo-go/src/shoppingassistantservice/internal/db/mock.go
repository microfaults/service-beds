package db

import "context"

type MockStore struct{}

func NewMockStore() *MockStore {
	return &MockStore{}
}

func (m *MockStore) SearchProducts(ctx context.Context, embedding []float32) ([]Product, error) {
	// Return hardcoded products
	return []Product{
		{ID: "OLJCESPC7Z", Description: "Vintage Camera Lens"},
		{ID: "66VCHSJNUP", Description: "Vintage Camera"},
		{ID: "1YMWWN1N4O", Description: "Home Barista Kit"},
		{ID: "L9ECAV7KIM", Description: "Terrarium"},
		{ID: "2REM4M0ZLS", Description: "Plants"},
	}, nil
}
