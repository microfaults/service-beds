package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"checkoutservice/models"
)

func (cs *checkoutService) getUserCart(userID string) ([]*models.CartItem, error) {
	url := fmt.Sprintf("%s/cart/%s", cs.cartSvcAddr, userID)

	resp, err := cs.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cart during checkout: %+v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user cart: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var cart models.Cart
	if err := json.NewDecoder(resp.Body).Decode(&cart); err != nil {
		return nil, fmt.Errorf("failed to decode cart response: %+v", err)
	}

	return cart.GetItems(), nil
}

func (cs *checkoutService) emptyUserCart(userID string) error {
	url := fmt.Sprintf("%s/cart/%s", cs.cartSvcAddr, userID)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create empty cart request: %+v", err)
	}

	resp, err := cs.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to empty user cart during checkout: %+v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to empty user cart: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

func (cs *checkoutService) getProduct(productID string) (*models.Product, error) {
	url := fmt.Sprintf("%s/product/%s", cs.productCatalogSvcAddr, productID)

	resp, err := cs.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %+v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get product: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var product models.Product
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		return nil, fmt.Errorf("failed to decode product response: %+v", err)
	}

	return &product, nil
}

func (cs *checkoutService) quoteShipping(address *models.Address, items []*models.CartItem) (*models.Money, error) {
	url := fmt.Sprintf("%s/quote", cs.shippingSvcAddr)

	reqBody := models.GetQuoteRequest{
		Address: address,
		Items:   items,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal quote request: %+v", err)
	}

	resp, err := cs.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to get shipping quote: %+v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get shipping quote: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var quoteResp models.GetQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&quoteResp); err != nil {
		return nil, fmt.Errorf("failed to decode quote response: %+v", err)
	}

	return quoteResp.GetCostUsd(), nil
}

func (cs *checkoutService) shipOrder(address *models.Address, items []*models.CartItem) (string, error) {
	url := fmt.Sprintf("%s/ship", cs.shippingSvcAddr)

	reqBody := models.ShipOrderRequest{
		Address: address,
		Items:   items,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ship order request: %+v", err)
	}

	resp, err := cs.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("shipment failed: %+v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("shipment failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var shipResp models.ShipOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&shipResp); err != nil {
		return "", fmt.Errorf("failed to decode ship response: %+v", err)
	}

	return shipResp.GetTrackingId(), nil
}

func (cs *checkoutService) convertCurrency(from *models.Money, toCurrency string) (*models.Money, error) {
	url := fmt.Sprintf("%s/convert", cs.currencySvcAddr)

	reqBody := models.CurrencyConversionRequest{
		From:   from,
		ToCode: toCurrency,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal currency conversion request: %+v", err)
	}

	resp, err := cs.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to convert currency: %+v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to convert currency: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result models.Money
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode currency conversion response: %+v", err)
	}

	return &result, nil
}

func (cs *checkoutService) chargeCard(amount *models.Money, paymentInfo *models.CreditCardInfo) (string, error) {
	url := fmt.Sprintf("%s/charge", cs.paymentSvcAddr)

	reqBody := models.ChargeRequest{
		Amount:     amount,
		CreditCard: paymentInfo,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal charge request: %+v", err)
	}

	resp, err := cs.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("could not charge the card: %+v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("could not charge the card: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var chargeResp models.ChargeResponse
	if err := json.NewDecoder(resp.Body).Decode(&chargeResp); err != nil {
		return "", fmt.Errorf("failed to decode charge response: %+v", err)
	}

	return chargeResp.GetTransactionId(), nil
}

func (cs *checkoutService) sendOrderConfirmation(email string, order *models.OrderResult) error {
	url := fmt.Sprintf("%s/send", cs.emailSvcAddr)

	reqBody := models.SendOrderConfirmationRequest{
		Email: email,
		Order: order,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal email request: %+v", err)
	}

	resp, err := cs.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to send order confirmation: %+v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send order confirmation: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}
