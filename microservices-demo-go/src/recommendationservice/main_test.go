package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListRecommendations(t *testing.T) {
	// Mock Product Catalog Service
	mockCatalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products" {
			t.Errorf("Expected request to /products, got %s", r.URL.Path)
		}
		products := ListProductsResponse{
			Products: []Product{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
				{ID: "4"},
				{ID: "5"},
				{ID: "6"}, // 6 products
			},
		}
		json.NewEncoder(w).Encode(products)
	}))
	defer mockCatalog.Close()

	// Initialize RecommendationService with nil pool (random fallback)
	svc := NewRecommendationService(
		strings.TrimPrefix(mockCatalog.URL, "http://"),
		nil, nil,
	)

	// Test Case 1: No exclusion
	t.Run("Returns 5 unique products", func(t *testing.T) {
		resp, err := svc.ListRecommendations(context.Background(), &ListRecommendationsRequest{
			UserID: "test-user",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.ProductIDs) != 5 {
			t.Errorf("Expected 5 products, got %d", len(resp.ProductIDs))
		}
	})

	// Test Case 2: Filter out products
	t.Run("Filters out products", func(t *testing.T) {
		resp, err := svc.ListRecommendations(context.Background(), &ListRecommendationsRequest{
			UserID:     "test-user",
			ProductIDs: []string{"1", "2"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Filtering 2 out of 6 leaves 4
		if len(resp.ProductIDs) != 4 {
			t.Errorf("Expected 4 products, got %d", len(resp.ProductIDs))
		}

		for _, id := range resp.ProductIDs {
			if id == "1" || id == "2" {
				t.Errorf("Expected product %s to be filtered out", id)
			}
		}
	})
}

func TestHandleListRecommendations_Integration(t *testing.T) {
	mockCatalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		products := ListProductsResponse{
			Products: []Product{
				{ID: "A"},
				{ID: "B"},
				{ID: "C"},
			},
		}
		json.NewEncoder(w).Encode(products)
	}))
	defer mockCatalog.Close()

	svc := NewRecommendationService(
		strings.TrimPrefix(mockCatalog.URL, "http://"),
		nil, nil,
	)

	reqBody, _ := json.Marshal(ListRecommendationsRequest{
		UserID:     "test-user",
		ProductIDs: []string{"A"},
	})
	req := httptest.NewRequest(http.MethodPost, "/recommendations", bytes.NewReader(reqBody))
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
		if id == "A" {
			t.Errorf("excluded product A should not appear")
		}
	}
}
