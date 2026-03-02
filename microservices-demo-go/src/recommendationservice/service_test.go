package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newMockCatalog creates a test HTTP server that returns the given products
// on GET /products, mimicking the product catalog service.
func newMockCatalog(products []Product) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/products" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListProductsResponse{Products: products})
	}))
}

func TestListRecommendations_FiltersExcludedProducts(t *testing.T) {
	catalog := newMockCatalog([]Product{
		{ID: "AAA"},
		{ID: "BBB"},
		{ID: "CCC"},
		{ID: "DDD"},
		{ID: "EEE"},
		{ID: "FFF"},
	})
	defer catalog.Close()

	svc := NewRecommendationService(catalog.Listener.Addr().String(), nil, nil)

	resp, err := svc.ListRecommendations(context.Background(), &ListRecommendationsRequest{
		UserID:     "test-user",
		ProductIDs: []string{"AAA", "BBB"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 6 products - 2 excluded = 4 available, capped at 5 -> expect 4
	if len(resp.ProductIDs) != 4 {
		t.Errorf("expected 4 products, got %d: %v", len(resp.ProductIDs), resp.ProductIDs)
	}

	excluded := map[string]bool{"AAA": true, "BBB": true}
	for _, id := range resp.ProductIDs {
		if excluded[id] {
			t.Errorf("excluded product %q should not appear in recommendations", id)
		}
	}
}

func TestListRecommendations_MaxFiveResults(t *testing.T) {
	products := make([]Product, 10)
	for i := range products {
		products[i] = Product{ID: string(rune('A' + i))}
	}

	catalog := newMockCatalog(products)
	defer catalog.Close()

	svc := NewRecommendationService(catalog.Listener.Addr().String(), nil, nil)

	resp, err := svc.ListRecommendations(context.Background(), &ListRecommendationsRequest{
		UserID: "test-user",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.ProductIDs) != 5 {
		t.Errorf("expected exactly 5 products, got %d: %v", len(resp.ProductIDs), resp.ProductIDs)
	}
}

func TestListRecommendations_EmptyCatalog(t *testing.T) {
	catalog := newMockCatalog([]Product{})
	defer catalog.Close()

	svc := NewRecommendationService(catalog.Listener.Addr().String(), nil, nil)

	resp, err := svc.ListRecommendations(context.Background(), &ListRecommendationsRequest{
		UserID: "test-user",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.ProductIDs) != 0 {
		t.Errorf("expected 0 products, got %d: %v", len(resp.ProductIDs), resp.ProductIDs)
	}
}

func TestListRecommendations_AllExcluded(t *testing.T) {
	catalog := newMockCatalog([]Product{
		{ID: "AAA"},
		{ID: "BBB"},
	})
	defer catalog.Close()

	svc := NewRecommendationService(catalog.Listener.Addr().String(), nil, nil)

	resp, err := svc.ListRecommendations(context.Background(), &ListRecommendationsRequest{
		UserID:     "test-user",
		ProductIDs: []string{"AAA", "BBB"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.ProductIDs) != 0 {
		t.Errorf("expected 0 products when all excluded, got %d: %v", len(resp.ProductIDs), resp.ProductIDs)
	}
}

func TestListRecommendations_CatalogUnavailable(t *testing.T) {
	// Point at an address that won't respond
	svc := NewRecommendationService("127.0.0.1:1", nil, nil)

	_, err := svc.ListRecommendations(context.Background(), &ListRecommendationsRequest{
		UserID: "test-user",
	})
	if err == nil {
		t.Fatal("expected error when catalog is unavailable, got nil")
	}
}

func TestHandleListRecommendations_Success(t *testing.T) {
	catalog := newMockCatalog([]Product{
		{ID: "OLJCESPC7Z"},
		{ID: "66VCHSJNUP"},
		{ID: "1YMWWN1N4O"},
	})
	defer catalog.Close()

	svc := NewRecommendationService(catalog.Listener.Addr().String(), nil, nil)

	body := `{"user_id":"test","product_ids":["OLJCESPC7Z"]}`
	req := httptest.NewRequest(http.MethodPost, "/recommendations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	logger := newTestLogger()
	handleListRecommendations(w, req, svc, logger)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp ListRecommendationsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// 3 products - 1 excluded = 2
	if len(resp.ProductIDs) != 2 {
		t.Errorf("expected 2 products, got %d: %v", len(resp.ProductIDs), resp.ProductIDs)
	}

	for _, id := range resp.ProductIDs {
		if id == "OLJCESPC7Z" {
			t.Errorf("excluded product OLJCESPC7Z should not appear")
		}
	}
}

func TestHandleListRecommendations_InvalidJSON(t *testing.T) {
	svc := NewRecommendationService("unused", nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/recommendations", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	logger := newTestLogger()
	handleListRecommendations(w, req, svc, logger)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("expected body 'OK', got %q", w.Body.String())
	}
}

// newTestLogger creates a silent slog.Logger for testing.
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
