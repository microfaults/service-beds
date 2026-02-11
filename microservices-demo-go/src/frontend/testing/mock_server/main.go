package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

// Mock Data Models
type Money struct {
	CurrencyCode string `json:"currency_code"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

type Product struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Picture     string   `json:"picture"`
	PriceUsd    *Money   `json:"price_usd"`
	Categories  []string `json:"categories"`
}

type CartItem struct {
	ProductId string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

type Order struct {
	OrderId      string `json:"order_id"`
	ShippingCost *Money `json:"shipping_cost"`
	Items        []*struct {
		Item *CartItem `json:"item"`
		Cost *Money    `json:"cost"`
	} `json:"items"`
}

type Ad struct {
	RedirectUrl string `json:"redirect_url"`
	Text        string `json:"text"`
}

func main() {
	r := mux.NewRouter()

	// --- Currency Service Mocks ---
	r.HandleFunc("/currencies", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"currency_codes": []string{"USD", "EUR", "CAD"},
		})
	}).Methods("GET")

	r.HandleFunc("/currency/convert", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			From   *Money `json:"from"`
			ToCode string `json:"to_code"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		// Mock conversion: 1:1
		json.NewEncoder(w).Encode(Money{
			CurrencyCode: req.ToCode,
			Units:        req.From.Units,
			Nanos:        req.From.Nanos,
		})
	}).Methods("POST")

	// --- Product Catalog Service Mocks ---
	products := []*Product{
		{
			Id:          "OLJCESPC7Z",
			Name:        "Mug",
			Description: "A simple mug with a mustard interior.",
			Picture:     "/static/img/products/mug.jpg", // typewriter.jpg missing, using mug.jpg
			PriceUsd:    &Money{CurrencyCode: "USD", Units: 67, Nanos: 990000000},
			Categories:  []string{"vintage"},
		},
		{
			Id:          "66VCHSJNUP",
			Name:        "Tank Top",
			Description: "This tank top is made from 100% organic cotton.",
			Picture:     "/static/img/products/tank-top.jpg",
			PriceUsd:    &Money{CurrencyCode: "USD", Units: 18, Nanos: 990000000},
			Categories:  []string{"clothing"},
		},
	}
	r.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"products": products})
	}).Methods("GET")

	r.HandleFunc("/products/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		for _, p := range products {
			if p.Id == id {
				json.NewEncoder(w).Encode(p)
				return
			}
		}
		http.Error(w, "Product not found", http.StatusNotFound)
	}).Methods("GET")

	// --- Cart Service Mocks ---
	// Simple in-memory cart
	cart := make(map[string][]*CartItem)

	r.HandleFunc("/cart/{userId}", func(w http.ResponseWriter, r *http.Request) {
		userId := mux.Vars(r)["userId"]
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": cart[userId],
		})
	}).Methods("GET")

	r.HandleFunc("/cart/{userId}/items", func(w http.ResponseWriter, r *http.Request) {
		userId := mux.Vars(r)["userId"]
		var item CartItem
		json.NewDecoder(r.Body).Decode(&item)
		cart[userId] = append(cart[userId], &item)
		w.WriteHeader(http.StatusOK)
	}).Methods("POST")

	r.HandleFunc("/cart/{userId}", func(w http.ResponseWriter, r *http.Request) {
		userId := mux.Vars(r)["userId"]
		delete(cart, userId)
		w.WriteHeader(http.StatusOK)
	}).Methods("DELETE")

	// --- Recommendation Service Mocks ---
	r.HandleFunc("/recommendations", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"product_ids": []string{"OLJCESPC7Z", "66VCHSJNUP"},
		})
	}).Methods("GET")

	// --- Shipping Service Mocks ---
	r.HandleFunc("/shipping/quote", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"cost_usd": Money{CurrencyCode: "USD", Units: 5, Nanos: 0},
		})
	}).Methods("POST")

	// --- Checkout Service Mocks ---
	r.HandleFunc("/checkout", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"order": Order{
				OrderId:      "mock-order-id-123",
				ShippingCost: &Money{CurrencyCode: "USD", Units: 5, Nanos: 0},
				Items: []*struct {
					Item *CartItem `json:"item"`
					Cost *Money    `json:"cost"`
				}{},
			},
		})
	}).Methods("POST")

	// --- Ad Service Mocks ---
	r.HandleFunc("/ads", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ads": []*Ad{
				{RedirectUrl: "/product/66VCHSJNUP", Text: "Ad: Check out our Tank Top!"},
			},
		})
	}).Methods("GET")

	// --- Shopping Assistant Mocks ---
	r.HandleFunc("/shopping-assistant", func(w http.ResponseWriter, r *http.Request) {
		// Mock response
	}).Methods("POST")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	fmt.Printf("Mock server listening on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
