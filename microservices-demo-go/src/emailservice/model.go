package main

// Money represents a monetary value
type Money struct {
	CurrencyCode string `json:"currency_code"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

// Address represents a shipping address
type Address struct {
	StreetAddress string `json:"street_address"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
	ZipCode       int32  `json:"zip_code"`
}

// CartItem represents an item in the cart
type CartItem struct {
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

// OrderItem represents an item in an order with its cost
type OrderItem struct {
	Item *CartItem `json:"item"`
	Cost *Money    `json:"cost"`
}

// OrderResult represents the result of a placed order
type OrderResult struct {
	OrderID            string       `json:"order_id"`
	ShippingTrackingID string       `json:"shipping_tracking_id"`
	ShippingCost       *Money       `json:"shipping_cost"`
	ShippingAddress    *Address     `json:"shipping_address"`
	Items              []*OrderItem `json:"items"`
}

// SendOrderConfirmationRequest represents the request payload for sending order confirmation
// Equivalent to Python: request.email and request.order
type SendOrderConfirmationRequest struct {
	Email string       `json:"email"`
	Order *OrderResult `json:"order"`
}

// HealthResponse represents the health check response
// Equivalent to Python: health_pb2.HealthCheckResponse
type HealthResponse struct {
	Status string `json:"status"`
}
