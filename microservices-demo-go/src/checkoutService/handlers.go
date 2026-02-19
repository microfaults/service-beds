package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"checkoutservice/models"
	"checkoutservice/money"

	"github.com/google/uuid"
)

// handleHealth handles health check requests
func (cs *checkoutService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "SERVING"})
}

// handlePlaceOrder handles order placement requests
func (cs *checkoutService) handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Errorf("failed to decode request: %v", err)
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Infof("[PlaceOrder] user_id=%q user_currency=%q", req.UserID, req.UserCurrency)

	orderID, err := uuid.NewUUID()
	if err != nil {
		log.Errorf("failed to generate order uuid: %v", err)
		http.Error(w, "failed to generate order uuid", http.StatusInternalServerError)
		return
	}

	prep, err := cs.prepareOrderItemsAndShippingQuoteFromCart(req.UserID, req.UserCurrency, req.Address)
	if err != nil {
		log.Errorf("failed to prepare order: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	total := models.Money{
		CurrencyCode: req.UserCurrency,
		Units:        0,
		Nanos:        0,
	}
	total = money.Must(money.Sum(total, *prep.shippingCostLocalized))
	for _, it := range prep.orderItems {
		multPrice := money.MultiplySlow(*it.Cost, uint32(it.Item.GetQuantity()))
		total = money.Must(money.Sum(total, multPrice))
	}

	txID, err := cs.chargeCard(&total, req.CreditCard)
	if err != nil {
		log.Errorf("failed to charge card: %v", err)
		http.Error(w, fmt.Sprintf("failed to charge card: %v", err), http.StatusInternalServerError)
		return
	}
	log.Infof("payment went through (transaction_id: %s)", txID)

	shippingTrackingID, err := cs.shipOrder(req.Address, prep.cartItems)
	if err != nil {
		log.Errorf("shipping error: %v", err)
		http.Error(w, fmt.Sprintf("shipping error: %v", err), http.StatusServiceUnavailable)
		return
	}

	_ = cs.emptyUserCart(req.UserID)

	orderResult := &models.OrderResult{
		OrderID:            orderID.String(),
		ShippingTrackingID: shippingTrackingID,
		ShippingCost:       prep.shippingCostLocalized,
		ShippingAddress:    req.Address,
		Items:              prep.orderItems,
	}

	if err := cs.sendOrderConfirmation(req.Email, orderResult); err != nil {
		log.Warnf("failed to send order confirmation to %q: %+v", req.Email, err)
	} else {
		log.Infof("order confirmation email sent to %q", req.Email)
	}

	resp := &models.PlaceOrderResponse{Order: orderResult}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
