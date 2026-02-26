package models

// Money represents a monetary value
type Money struct {
	CurrencyCode string `json:"currency_code"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

// GetCurrencyCode returns the currency code
func (m Money) GetCurrencyCode() string { return m.CurrencyCode }

// GetUnits returns the units
func (m Money) GetUnits() int64 { return m.Units }

// GetNanos returns the nanos
func (m Money) GetNanos() int32 { return m.Nanos }

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

type CreditCardInfo struct {
	CreditCardNumber          string `json:"credit_card_number"`
	CreditCardCVV             int32  `json:"credit_card_cvv"`
	CreditCardExpirationYear  int32  `json:"credit_card_expiration_year"`
	CreditCardExpirationMonth int32  `json:"credit_card_expiration_month"`
}

type PlaceOrderRequest struct {
	UserID       string          `json:"user_id"`
	UserCurrency string          `json:"user_currency"`
	Address      *Address        `json:"address"`
	Email        string          `json:"email"`
	CreditCard   *CreditCardInfo `json:"credit_card"`
}

type PlaceOrderResponse struct {
	Order *OrderResult `json:"order"`
}

// GetQuantity returns the quantity of the cart item
func (c *CartItem) GetQuantity() int32 {
	if c == nil {
		return 0
	}
	return c.Quantity
}

// GetProductId returns the product ID of the cart item
func (c *CartItem) GetProductId() string {
	if c == nil {
		return ""
	}
	return c.ProductID
}

// Product represents a product in the catalog
type Product struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Picture     string   `json:"picture"`
	PriceUsd    *Money   `json:"price_usd"`
	Categories  []string `json:"categories"`
}

// GetPriceUsd returns the price in USD
func (p *Product) GetPriceUsd() *Money {
	if p == nil {
		return nil
	}
	return p.PriceUsd
}

// Cart represents a user's shopping cart
type Cart struct {
	UserID string      `json:"user_id"`
	Items  []*CartItem `json:"items"`
}

// GetItems returns the items in the cart
func (c *Cart) GetItems() []*CartItem {
	if c == nil {
		return nil
	}
	return c.Items
}

// GetQuoteRequest represents a shipping quote request
type GetQuoteRequest struct {
	Address *Address    `json:"address"`
	Items   []*CartItem `json:"items"`
}

// GetQuoteResponse represents a shipping quote response
type GetQuoteResponse struct {
	CostUsd *Money `json:"cost_usd"`
}

// GetCostUsd returns the shipping cost in USD
func (r *GetQuoteResponse) GetCostUsd() *Money {
	if r == nil {
		return nil
	}
	return r.CostUsd
}

// ShipOrderRequest represents a ship order request
type ShipOrderRequest struct {
	Address *Address    `json:"address"`
	Items   []*CartItem `json:"items"`
}

// ShipOrderResponse represents a ship order response
type ShipOrderResponse struct {
	TrackingID string `json:"tracking_id"`
}

// GetTrackingId returns the tracking ID
func (r *ShipOrderResponse) GetTrackingId() string {
	if r == nil {
		return ""
	}
	return r.TrackingID
}

// CurrencyConversionRequest represents a currency conversion request
type CurrencyConversionRequest struct {
	From   *Money `json:"from"`
	ToCode string `json:"to_code"`
}

// ChargeRequest represents a payment charge request
type ChargeRequest struct {
	Amount     *Money          `json:"amount"`
	CreditCard *CreditCardInfo `json:"credit_card"`
}

// ChargeResponse represents a payment charge response
type ChargeResponse struct {
	TransactionID string `json:"transaction_id"`
}

// GetTransactionId returns the transaction ID
func (r *ChargeResponse) GetTransactionId() string {
	if r == nil {
		return ""
	}
	return r.TransactionID
}

// SendOrderConfirmationRequest represents an email confirmation request
type SendOrderConfirmationRequest struct {
	Email string       `json:"email"`
	Order *OrderResult `json:"order"`
}
