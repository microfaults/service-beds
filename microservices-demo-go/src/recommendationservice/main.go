package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"git.ucsc.edu/microfaults/atropos-go"
)

type Product struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProductList struct {
	Products []Product `json:"products"`
}

type RecommendationRequest struct {
	ProductIDs []string `json:"product_ids"`
}

type RecommendationResponse struct {
	ProductIDs []string `json:"product_ids"`
}

type RecommendationService struct {
	Client      *http.Client
	CatalogAddr string
}

func (s *RecommendationService) ListRecommendations(w http.ResponseWriter, r *http.Request) {
	var req RecommendationRequest
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// It's optional to have a body, so just ignore error but maybe log it?
			// For now, let's assume empty request if decode fails
			req.ProductIDs = []string{}
		}
	}

	// Fetch products from product catalog
	url := fmt.Sprintf("http://%s/products", s.CatalogAddr)
	resp, err := s.Client.Get(url)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch products: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("failed to fetch products: status %d", resp.StatusCode), http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read response body: %v", err), http.StatusInternalServerError)
		return
	}

	var productList ProductList
	if err := json.Unmarshal(body, &productList); err != nil {
		http.Error(w, fmt.Sprintf("failed to unmarshal products: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter out products that are already in the request
	filteredProducts := []string{}
	existingIDs := make(map[string]bool)
	for _, id := range req.ProductIDs {
		existingIDs[id] = true
	}

	for _, p := range productList.Products {
		if !existingIDs[p.ID] {
			filteredProducts = append(filteredProducts, p.ID)
		}
	}

	// Shuffle and pick 5
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(filteredProducts), func(i, j int) { filteredProducts[i], filteredProducts[j] = filteredProducts[j], filteredProducts[i] })

	count := 5
	if len(filteredProducts) < count {
		count = len(filteredProducts)
	}
	selection := filteredProducts[:count]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RecommendationResponse{ProductIDs: selection})
}

func (s *RecommendationService) Check(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "SERVING"})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()

	catalogAddr := os.Getenv("PRODUCT_CATALOG_SERVICE_ADDR")
	if catalogAddr == "" {
		catalogAddr = "productcatalogservice:3550"
	}

	svc := &RecommendationService{
		Client: &http.Client{
			Transport: atropos.EgressTransport(http.DefaultTransport),
			Timeout:   10 * time.Second,
		},
		CatalogAddr: catalogAddr,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/recommendations", svc.ListRecommendations)
	mux.HandleFunc("/_healthz", svc.Check)

	h, shutdown, err := atropos.Serve(ctx, atropos.Config{
		Service: "recommendationservice",
		Version: "0.1.0",
		Routes: []atropos.Route{
			{Method: "GET", Path: "/recommendations", Description: "List product recommendations (query: product_ids, user_id)"},
		},
		Handler: mux,
	})
	if err != nil {
		log.Fatalf("failed to init atropos: %v", err)
	}
	defer shutdown(ctx)

	log.Printf("recommendationservice listening on port %s", port)
	if err := http.ListenAndServe(":"+port, h); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
