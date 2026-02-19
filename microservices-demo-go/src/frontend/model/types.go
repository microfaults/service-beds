package model

type Money struct {
	CurrencyCode string `json:"currency_code"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

type Product struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Picture     string   `json:"picture"`
	PriceUsd    *Money   `json:"price_usd"`
	Categories  []string `json:"categories"`
}

type CartItem struct {
	ProductId string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

type Address struct {
	StreetAddress string `json:"street_address"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
	ZipCode       string  `json:"zip_code"`
}

type OrderItem struct {
	Item *CartItem `json:"item"`
	Cost *Money    `json:"cost"`
}

type Order struct {
	OrderId            string       `json:"order_id"`
	ShippingTrackingId string       `json:"shipping_tracking_id"`
	ShippingCost       *Money       `json:"shipping_cost"`
	ShippingAddress    *Address     `json:"shipping_address"`
	Items              []*OrderItem `json:"items"`
}

type CreditCardInfo struct {
	CreditCardNumber          string `json:"credit_card_number"`
	CreditCardExpirationMonth int32  `json:"expiry_month"`
	CreditCardExpirationYear  int32  `json:"expiry_year"`
	CreditCardCvv             int32  `json:"cvv"`
}

type PlaceOrderRequest struct {
	Email        string          `json:"email"`
	CreditCard   *CreditCardInfo `json:"credit_card"`
	UserId       string          `json:"user_id"`
	UserCurrency string          `json:"user_currency"`
	Address      *Address        `json:"address"`
}

type PlaceOrderResponse struct {
	Order *Order `json:"order"`
}

type Ad struct {
	RedirectUrl string `json:"redirect_url"`
	Text        string `json:"text"`
}

type AdRequest struct {
	ContextKeys []string `json:"context_keys"`
}

type AdResponse struct {
	Ads []*Ad `json:"ads"`
}
