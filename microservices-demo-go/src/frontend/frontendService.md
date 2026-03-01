# Frontend Service Entry Points → Downstream Services

## Summary
The **Frontend** service is the primary HTTP entrypoint for the web application.  
It serves HTML templates and static assets while orchestrating calls to backend microservices.

It communicates with downstream services for:
- Product data
- Currency conversion
- Cart management
- Recommendations
- Shipping quotes
- Checkout processing
- Advertisements
- AI assistant interactions

This service acts as the system’s main ingress point.

---

## APIs / Entry Points

| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| GET, HEAD | `/` | Currency, Product Catalog, Cart, Ad |
| GET, HEAD | `/product/{id}` | Product Catalog, Currency, Cart, Recommendation, Ad |
| GET, HEAD | `/cart` | Currency, Cart, Product Catalog, Recommendation, Shipping |
| POST | `/cart` | Product Catalog, Cart |
| POST | `/cart/empty` | Cart |
| POST | `/cart/checkout` | Checkout |
| POST | `/setCurrency` | None |
| GET | `/logout` | None |
| GET | `/assistant` | Currency |
| GET | `/product-meta/{ids}` | Product Catalog |
| POST | `/bot` | Shopping Assistant |
| GET | `/robots.txt` | None |
| GET | `/_healthz` | None |
| GET | `/static/*` | None (local filesystem) |