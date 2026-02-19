package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/microservices-demo-go/src/frontend/model"
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type CurrencyClient interface {
	GetSupportedCurrencies(ctx context.Context) ([]string, error)
	Convert(ctx context.Context, from *model.Money, toCurrency string) (*model.Money, error)
}

type ProductCatalogClient interface {
	ListProducts(ctx context.Context) ([]*model.Product, error)
	GetProduct(ctx context.Context, id string) (*model.Product, error)
}

type CartClient interface {
	GetCart(ctx context.Context, userID string) ([]*model.CartItem, error)
	AddItem(ctx context.Context, userID, productID string, quantity int32) error
	EmptyCart(ctx context.Context, userID string) error
}

type RecommendationClient interface {
	ListRecommendations(ctx context.Context, userID string, productIDs []string) ([]*model.Product, error)
}

type ShippingClient interface {
	GetQuote(ctx context.Context, items []*model.CartItem, currency string) (*model.Money, error)
}

type CheckoutClient interface {
	PlaceOrder(ctx context.Context, req *model.PlaceOrderRequest) (*model.Order, error)
}

type AdClient interface {
	GetAds(ctx context.Context, contextKeys []string) ([]*model.Ad, error)
}

// HTTP implementation
type httpCurrencyClient struct {
	addr   string
	client *http.Client
}

func NewCurrencyClient(addr string) CurrencyClient {
	return &httpCurrencyClient{
		addr:   addr,
		client: &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
	}
}

func (c *httpCurrencyClient) GetSupportedCurrencies(ctx context.Context) ([]string, error) {
	resp, err := c.client.Get(fmt.Sprintf("http://%s/currencies", c.addr))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		CurrencyCodes []string `json:"currency_codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.CurrencyCodes, nil
}

func (c *httpCurrencyClient) Convert(ctx context.Context, from *model.Money, toCurrency string) (*model.Money, error) {
	reqBody := struct {
		From   *model.Money `json:"from"`
		ToCode string       `json:"to_code"`
	}{
		From:   from,
		ToCode: toCurrency,
	}
	b, _ := json.Marshal(reqBody)
	resp, err := c.client.Post(fmt.Sprintf("http://%s/convert", c.addr), "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("currency conversion failed: status %d", resp.StatusCode)
	}
	var out model.Money
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

type httpProductCatalogClient struct {
	addr   string
	client *http.Client
}

func NewProductCatalogClient(addr string) ProductCatalogClient {
	return &httpProductCatalogClient{
		addr:   addr,
		client: &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
	}
}

func (c *httpProductCatalogClient) ListProducts(ctx context.Context) ([]*model.Product, error) {
	resp, err := c.client.Get(fmt.Sprintf("http://%s/products", c.addr))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for non-200 status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list products: status %d", resp.StatusCode)
	}

	var out struct {
		Products []*model.Product `json:"products"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, errors.Wrap(err, "failed to decode products")
	}
	return out.Products, nil
}

func (c *httpProductCatalogClient) GetProduct(ctx context.Context, id string) (*model.Product, error) {
	resp, err := c.client.Get(fmt.Sprintf("http://%s/products/%s", c.addr, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get product %s: status %d", id, resp.StatusCode)
	}
	var out model.Product
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

type httpCartClient struct {
	addr   string
	client *http.Client
}

func NewCartClient(addr string) CartClient {
	return &httpCartClient{
		addr:   addr,
		client: &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
	}
}

func (c *httpCartClient) GetCart(ctx context.Context, userID string) ([]*model.CartItem, error) {
	resp, err := c.client.Get(fmt.Sprintf("http://%s/cart/%s", c.addr, userID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Items []*model.CartItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *httpCartClient) AddItem(ctx context.Context, userID, productID string, quantity int32) error {
	reqBody := model.CartItem{
		ProductId: productID,
		Quantity:  quantity,
	}
	reqBodyFix := struct {
		Item *model.CartItem `json:"item"`
	}{
		Item: &reqBody,
	}
	fmt.Println("Error adding item to cart:", reqBodyFix)
	b, _ := json.Marshal(reqBodyFix)
	resp, err := c.client.Post(fmt.Sprintf("http://%s/cart/%s/items", c.addr, userID), "application/json", bytes.NewReader(b))
	fmt.Println("Error adding item to cart:", resp)
	if err != nil {
		fmt.Println("Error adding item to cart:", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to add item: status %d", resp.StatusCode)
	}
	return nil
}

func (c *httpCartClient) EmptyCart(ctx context.Context, userID string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://%s/cart/%s", c.addr, userID), nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to empty cart: status %d", resp.StatusCode)
	}
	return nil
}

type httpRecommendationClient struct {
	addr   string
	client *http.Client
	pc     ProductCatalogClient
}

func NewRecommendationClient(addr string, pc ProductCatalogClient) RecommendationClient {
	return &httpRecommendationClient{
		addr:   addr,
		client: &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
		pc:     pc,
	}
}

func (c *httpRecommendationClient) ListRecommendations(ctx context.Context, userID string, productIDs []string) ([]*model.Product, error) {
	// Query params for productIDs? Or POST?
	// Assuming GET /recommendations?product_ids=...&user_id=...
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/recommendations", c.addr), nil)
	q := req.URL.Query()
	q.Add("user_id", userID)
	for _, pid := range productIDs {
		q.Add("product_ids", pid)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("recommendation service returned %d", resp.StatusCode)
	}

	var out struct {
		ProductIds []string `json:"product_ids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		// Try decoding generic list if needed, or assume struct wrapper
		return nil, err
	}

	// Fetch actual products from product catalog
	var products []*model.Product
	for _, id := range out.ProductIds {
		if p, err := c.pc.GetProduct(ctx, id); err == nil {
			products = append(products, p)
		}
	}
	return products, nil
}

type httpShippingClient struct {
	addr   string
	client *http.Client
}

func NewShippingClient(addr string) ShippingClient {
	return &httpShippingClient{
		addr:   addr,
		client: &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
	}
}

func (c *httpShippingClient) GetQuote(ctx context.Context, items []*model.CartItem, currency string) (*model.Money, error) {
	reqBody := struct {
		Items    []*model.CartItem `json:"items"`
		Currency string            `json:"currency"` // Assuming shipping service can handle currency, otherwise consumer converts?
	}{
		Items:    items,
		Currency: currency,
	}
	// The original gRPC GetQuoteRequest has Address and Items.
	// We should probably check the original method again.
	// rpc.go: GetQuoteRequest{Address: nil, Items: items}

	b, _ := json.Marshal(reqBody)
	resp, err := c.client.Post(fmt.Sprintf("http://%s/shipping/quote", c.addr), "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		CostUsd *model.Money `json:"cost_usd"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.CostUsd, nil // Consumer (frontend) will convert currency
}

type httpCheckoutClient struct {
	addr   string
	client *http.Client
}

func NewCheckoutClient(addr string) CheckoutClient {
	return &httpCheckoutClient{
		addr:   addr,
		client: &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
	}
}

func (c *httpCheckoutClient) PlaceOrder(ctx context.Context, req *model.PlaceOrderRequest) (*model.Order, error) {
	b, _ := json.Marshal(req)
	resp, err := c.client.Post(fmt.Sprintf("http://%s/placeorder", c.addr), "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("checkout failed: %d", resp.StatusCode)
	}
	var out model.PlaceOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Order, nil
}

type httpAdClient struct {
	addr   string
	client *http.Client
}

func NewAdClient(addr string) AdClient {
	return &httpAdClient{
		addr:   addr,
		client: &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
	}
}

func (c *httpAdClient) GetAds(ctx context.Context, contextKeys []string) ([]*model.Ad, error) {
	// GET /ads?context_keys=...
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/ads", c.addr), nil)
	q := req.URL.Query()
	for _, k := range contextKeys {
		q.Add("context_keys", k)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out model.AdResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Ads, nil
}
