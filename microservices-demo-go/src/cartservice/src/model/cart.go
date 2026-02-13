package model

type CartItem struct {
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

type Cart struct {
	UserID string      `json:"user_id"`
	Items  []*CartItem `json:"items"`
}

// AddItem helper to easily manage items logic
func (c *Cart) AddItem(productID string, quantity int32) {
	for _, item := range c.Items {
		if item.ProductID == productID {
			item.Quantity += quantity
			return
		}
	}
	c.Items = append(c.Items, &CartItem{
		ProductID: productID,
		Quantity:  quantity,
	})
}
