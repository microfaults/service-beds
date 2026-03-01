# Product Catalog Service Entry Points → Downstream Services

## Summary
An HTTP service that manages and serves product data for the storefront.

It supports:
- Listing all products
- Fetching a single product by ID
- Searching products by query string
- Fetching multiple products in a batch request

---

## APIs / Entry Points

| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| GET | `/products` | None |
| POST | `/products/batch` | None |
| GET | `/products/{id}` | None |
| GET | `/products/search` | None |
| GET | `/_healthz` | None |