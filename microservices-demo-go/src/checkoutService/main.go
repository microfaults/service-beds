package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"checkoutservice/models"
	"checkoutservice/money"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	listenPort  = "5050"
	usdCurrency = "USD"
)

var log *logrus.Logger

func init() {
	log = logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout
}

type checkoutService struct {
	productCatalogSvcAddr string
	cartSvcAddr           string
	currencySvcAddr       string
	shippingSvcAddr       string
	emailSvcAddr          string
	paymentSvcAddr        string
	httpClient            *http.Client
}

func main() {
	port := listenPort
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	svc := &checkoutService{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	mustMapEnv(&svc.shippingSvcAddr, "SHIPPING_SERVICE_ADDR")
	mustMapEnv(&svc.productCatalogSvcAddr, "PRODUCT_CATALOG_SERVICE_ADDR")
	mustMapEnv(&svc.cartSvcAddr, "CART_SERVICE_ADDR")
	mustMapEnv(&svc.currencySvcAddr, "CURRENCY_SERVICE_ADDR")
	mustMapEnv(&svc.emailSvcAddr, "EMAIL_SERVICE_ADDR")
	mustMapEnv(&svc.paymentSvcAddr, "PAYMENT_SERVICE_ADDR")

	log.Infof("service config: %+v", svc)

	// Set up HTTP routes
	http.HandleFunc("/placeorder", svc.handlePlaceOrder)
	http.HandleFunc("/health", svc.handleHealth)

	log.Infof("starting to listen on http://:%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatal(err)
	}
}

func mustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		panic(fmt.Sprintf("environment variable %q not set", envKey))
	}
	*target = v
}

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

type orderPrep struct {
	orderItems            []*models.OrderItem
	cartItems             []*models.CartItem
	shippingCostLocalized *models.Money
}

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
