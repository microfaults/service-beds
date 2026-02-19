package main

import (
	"fmt"

	"checkoutservice/models"
)

func (cs *checkoutService) prepareOrderItemsAndShippingQuoteFromCart(userID, userCurrency string, address *models.Address) (orderPrep, error) {
	var out orderPrep

	cartItems, err := cs.getUserCart(userID)
	if err != nil {
		return out, fmt.Errorf("cart failure: %+v", err)
	}

	orderItems, err := cs.prepOrderItems(cartItems, userCurrency)
	if err != nil {
		return out, fmt.Errorf("failed to prepare order: %+v", err)
	}

	shippingUSD, err := cs.quoteShipping(address, cartItems)
	if err != nil {
		return out, fmt.Errorf("shipping quote failure: %+v", err)
	}

	shippingPrice, err := cs.convertCurrency(shippingUSD, userCurrency)
	if err != nil {
		return out, fmt.Errorf("failed to convert shipping cost to currency: %+v", err)
	}

	out.shippingCostLocalized = shippingPrice
	out.cartItems = cartItems
	out.orderItems = orderItems
	return out, nil
}

func (cs *checkoutService) prepOrderItems(items []*models.CartItem, userCurrency string) ([]*models.OrderItem, error) {
	out := make([]*models.OrderItem, len(items))

	for i, item := range items {
		product, err := cs.getProduct(item.GetProductId())
		if err != nil {
			return nil, fmt.Errorf("failed to get product #%q", item.GetProductId())
		}

		price, err := cs.convertCurrency(product.GetPriceUsd(), userCurrency)
		if err != nil {
			return nil, fmt.Errorf("failed to convert price of %q to %s", item.GetProductId(), userCurrency)
		}

		out[i] = &models.OrderItem{
			Item: item,
			Cost: price,
		}
	}

	return out, nil
}
