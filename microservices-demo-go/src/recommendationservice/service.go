package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const maxResponses = 5

// RecommendationService implements the recommendation business logic.
type RecommendationService struct {
	catalogAddr  string
	httpClient   *http.Client
	popularityDB *pgxpool.Pool
	logger       *slog.Logger
}

// The catalogAddr should be the host:port of the product catalog service (e.g. "productcatalogservice:3550").
// The pool may be nil, in which case the service falls back to random recommendations.
func NewRecommendationService(catalogAddr string, pool *pgxpool.Pool, logger *slog.Logger) *RecommendationService {
	if logger == nil {
		logger = slog.Default()
	}
	return &RecommendationService{
		catalogAddr:  catalogAddr,
		popularityDB: pool,
		logger:       logger,
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

	// 4. Try popularity-based ordering; fall back to random sampling
	prodList := s.sortByPopularity(ctx, filtered)

	numReturn := min(maxResponses, len(prodList))
	prodList = prodList[:numReturn]

	return &ListRecommendationsResponse{
		ProductIDs: prodList,
	}, nil
}

// sortByPopularity queries the popularity database for checkout counts and
// returns the filtered products sorted by descending popularity. If the
// database is unavailable or has no data, it falls back to random ordering.
func (s *RecommendationService) sortByPopularity(ctx context.Context, filtered []string) []string {
	if s.popularityDB == nil || len(filtered) == 0 {
		return randomSample(filtered)
	}

	rows, err := s.popularityDB.Query(ctx,
		`SELECT product_id, total_quantity FROM checkout_counts ORDER BY total_quantity DESC`)
	if err != nil {
		s.logger.Warn("failed to query popularity database, falling back to random", "error", err)
		return randomSample(filtered)
	}
	defer rows.Close()

	// Build a map of product_id -> rank (lower rank = more popular)
	popularityRank := make(map[string]int)
	rank := 0
	for rows.Next() {
		var productID string
		var totalQty int64
		if err := rows.Scan(&productID, &totalQty); err != nil {
			s.logger.Warn("failed to scan popularity row", "error", err)
			continue
		}
		popularityRank[productID] = rank
		rank++
	}
	if err := rows.Err(); err != nil {
		s.logger.Warn("error iterating popularity rows, falling back to random", "error", err)
		return randomSample(filtered)
	}

	if len(popularityRank) == 0 {
		return randomSample(filtered)
	}

	// Sort filtered products: those with popularity data come first (by rank),
	// those without come after in their original order.
	result := make([]string, len(filtered))
	copy(result, filtered)

	sort.SliceStable(result, func(i, j int) bool {
		ri, okI := popularityRank[result[i]]
		rj, okJ := popularityRank[result[j]]
		if okI && okJ {
			return ri < rj
		}
		if okI {
			return true
		}
		return false
	})

	return result
}

// randomSample returns a randomly shuffled copy of the input slice.
func randomSample(items []string) []string {
	result := make([]string, len(items))
	copy(result, items)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result
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
