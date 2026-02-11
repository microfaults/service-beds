package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const maxResponses = 5

// RecommendationService implements the recommendation business logic.
type RecommendationService struct {
	catalogAddr string
	httpClient  *http.Client
}

// NewRecommendationService creates a new RecommendationService.
// The catalogAddr should be the host:port of the product catalog service (e.g. "productcatalogservice:3550").
func NewRecommendationService(catalogAddr string) *RecommendationService {
	return &RecommendationService{
		catalogAddr: catalogAddr,
		httpClient: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

// ListRecommendations fetches all products from the product catalog service,
// filters out the ones already in the request, and returns a random sample of up to 5.
// Equivalent to Python: RecommendationService.ListRecommendations
func (s *RecommendationService) ListRecommendations(ctx context.Context, req *ListRecommendationsRequest) (*ListRecommendationsResponse, error) {
	// 1. Fetch list of products from product catalog service
	products, err := s.fetchProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}

	// 2. Extract product IDs
	productIDs := make([]string, len(products))
	for i, p := range products {
		productIDs[i] = p.ID
	}

	// 3. Filter out products already in the request using a set
	excludeSet := make(map[string]struct{}, len(req.ProductIDs))
	for _, id := range req.ProductIDs {
		excludeSet[id] = struct{}{}
	}

	var filtered []string
	for _, id := range productIDs {
		if _, excluded := excludeSet[id]; !excluded {
			filtered = append(filtered, id)
		}
	}

	// 4. Sample up to maxResponses from the filtered list
	numReturn := min(maxResponses, len(filtered))

	// Sample random indices
	indices := rand.Perm(len(filtered))[:numReturn]
	prodList := make([]string, numReturn)
	for i, idx := range indices {
		prodList[i] = filtered[idx]
	}

	return &ListRecommendationsResponse{
		ProductIDs: prodList,
	}, nil
}

// fetchProducts calls the product catalog service's GET /products endpoint.
func (s *RecommendationService) fetchProducts(ctx context.Context) ([]Product, error) {
	url := fmt.Sprintf("http://%s/products", s.catalogAddr)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call product catalog service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product catalog service returned status %d", resp.StatusCode)
	}

	var listResp ListProductsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode product catalog response: %w", err)
	}

	return listResp.Products, nil
}
