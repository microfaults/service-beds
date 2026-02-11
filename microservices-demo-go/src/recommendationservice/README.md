# Recommendation Service (Go)

A Go HTTP/REST transcompilation of the original Python gRPC recommendation service. It recommends products by fetching the full catalog from the Product Catalog Service, filtering out products the user has already viewed, and returning a random sample of up to 5 product IDs.

## Prerequisites

- **Go 1.25+** installed ([download](https://go.dev/dl/))
- **Docker** (optional, for containerised builds)

## Running the Tests

From this directory:

```bash
cd service-beds/microservices-demo-go/src/recommendationservice
go test -v ./...
```

### What the tests cover

| Test | Description |
|------|-------------|
| `TestListRecommendations_FiltersExcludedProducts` | Verifies that products in the request's `product_ids` are excluded from results |
| `TestListRecommendations_MaxFiveResults` | Confirms that at most 5 products are returned even when many are available |
| `TestListRecommendations_EmptyCatalog` | Returns an empty list when the catalog has no products |
| `TestListRecommendations_AllExcluded` | Returns an empty list when every catalog product is excluded |
| `TestListRecommendations_CatalogUnavailable` | Returns an error when the upstream catalog service is unreachable |
| `TestHandleListRecommendations_Success` | End-to-end HTTP handler test with a mock catalog |
| `TestHandleListRecommendations_InvalidJSON` | Returns 400 Bad Request for malformed JSON input |
| `TestHealthEndpoint` | Checks that `GET /health` returns `200 OK` |

### Running a single test

```bash
go test -v -run TestListRecommendations_MaxFiveResults ./...
```

### Running with the race detector

```bash
go test -race -v ./...
```

## Running the Service Locally

```bash
# Required
export PRODUCT_CATALOG_SERVICE_ADDR="localhost:3550"

# Optional (defaults shown)
export PORT="8080"
export ENABLE_TRACING="0"
export COLLECTOR_SERVICE_ADDR="localhost:4317"

go run .
```

### Example request

```bash
curl -X POST http://localhost:8080/recommendations \
  -H "Content-Type: application/json" \
  -d '{"user_id": "abc123", "product_ids": ["OLJCESPC7Z"]}'
```

### Health check

```bash
curl http://localhost:8080/health
```

## Building with Docker

```bash
docker build -t recommendationservice .
docker run -e PRODUCT_CATALOG_SERVICE_ADDR=host.docker.internal:3550 -p 8080:8080 recommendationservice
```
