package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestListRecommendations(t *testing.T) {
	// Mock Product Catalog Service
	mockCatalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products" {
			t.Errorf("Expected request to /products, got %s", r.URL.Path)
		}
		products := ProductList{
			Products: []Product{
				{ID: "1", Name: "Product 1"},
				{ID: "2", Name: "Product 2"},
				{ID: "3", Name: "Product 3"},
				{ID: "4", Name: "Product 4"},
				{ID: "5", Name: "Product 5"},
				{ID: "6", Name: "Product 6"}, // 6 products
			},
		}
		json.NewEncoder(w).Encode(products)
	}))
	defer mockCatalog.Close()

	// Initialize RecommendationService
	svc := &RecommendationService{
		Client:      &http.Client{Timeout: 5 * time.Second},
		CatalogAddr: strings.TrimPrefix(mockCatalog.URL, "http://"),
	}

	// Test Case 1: No exclusion
	t.Run("Returns 5 unique products", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/recommendations", nil)
		w := httptest.NewRecorder()

		svc.ListRecommendations(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var recResp RecommendationResponse
		if err := json.NewDecoder(resp.Body).Decode(&recResp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(recResp.ProductIDs) != 5 {
			t.Errorf("Expected 5 products, got %d", len(recResp.ProductIDs))
		}
	})

	// Test Case 2: Filter out products
	t.Run("Filters out products", func(t *testing.T) {
		excludeIDs := []string{"1", "2"}
		reqBody := RecommendationRequest{ProductIDs: excludeIDs}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/recommendations", bytes.NewReader(bodyBytes))
		w := httptest.NewRecorder()

		svc.ListRecommendations(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var recResp RecommendationResponse
		if err := json.NewDecoder(resp.Body).Decode(&recResp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should receive 4 items: 3, 4, 5, 6 (since 1 and 2 are excluded and total is 6)
		// Wait, filtering 2 out of 6 leaves 4. So count should be 4.
		if len(recResp.ProductIDs) != 4 {
			t.Errorf("Expected 4 products, got %d", len(recResp.ProductIDs))
		}

		for _, id := range recResp.ProductIDs {
			if id == "1" || id == "2" {
				t.Errorf("Expected product %s to be filtered out", id)
			}
		}
	})
}
