package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerListProducts(t *testing.T) {
	setupMock()
	handler := &ProductHandler{service: mockProductCatalog}

	req, err := http.NewRequest("GET", "/products", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	http.HandlerFunc(handler.ListProducts).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response struct {
		Products []*Product `json:"products"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if len(response.Products) != 4 {
		t.Errorf("expected 4 products, got %v", len(response.Products))
	}
}

func TestHandlerGetProduct(t *testing.T) {
	setupMock()
	handler := &ProductHandler{service: mockProductCatalog}

	// Note: Standard library's PathValue requires the request to go through a Mux in tests
	mux := http.NewServeMux()
	mux.HandleFunc("GET /products/{id}", handler.GetProduct)

	req, err := http.NewRequest("GET", "/products/abc001", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var p Product
	if err := json.NewDecoder(rr.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}

	if p.Name != "Product Alpha One" {
		t.Errorf("expected Product Alpha One, got %s", p.Name)
	}
}

func TestHandlerSearchProducts(t *testing.T) {
	setupMock()
	handler := &ProductHandler{service: mockProductCatalog}

	req, err := http.NewRequest("GET", "/products/search?q=Delta", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	http.HandlerFunc(handler.SearchProducts).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response struct {
		Results []*Product `json:"results"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if len(response.Results) != 1 || response.Results[0].Name != "Product Delta" {
		t.Errorf("unexpected search results: %+v", response.Results)
	}
}
