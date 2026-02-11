package main

// ListRecommendationsRequest represents the incoming request body.
// Equivalent to proto: hipstershop.ListRecommendationsRequest
type ListRecommendationsRequest struct {
	UserID     string   `json:"user_id"`
	ProductIDs []string `json:"product_ids"`
}

// ListRecommendationsResponse represents the outgoing response body.
// Equivalent to proto: hipstershop.ListRecommendationsResponse
type ListRecommendationsResponse struct {
	ProductIDs []string `json:"product_ids"`
}

// Product represents a product from the product catalog service.
// Only the ID field is needed for recommendation logic.
type Product struct {
	ID string `json:"id"`
}

// ListProductsResponse represents the response from the product catalog service's GET /products.
type ListProductsResponse struct {
	Products []Product `json:"products"`
}
