package main

import (
	"checkoutservice/models"
	"net/http"
)

type checkoutService struct {
	productCatalogSvcAddr string
	cartSvcAddr           string
	currencySvcAddr       string
	shippingSvcAddr       string
	emailSvcAddr          string
	paymentSvcAddr        string
	httpClient            *http.Client
}

type orderPrep struct {
	orderItems            []*models.OrderItem
	cartItems             []*models.CartItem
	shippingCostLocalized *models.Money
}
