# Service Matrix & Interconnect (HTTP) — Entry Points → Downstream Services

This document lists **each service**, a brief **summary**, its **HTTP entry points**, and the **downstream services** each entry point calls (service names only).  

---

## Frontend Entry Points → Downstream Services

### Summary
The **frontend** service is the HTTP entrypoint for the web app. It serves HTML + static assets and exposes user-facing routes (home, product pages, cart actions, checkout, etc.). It calls multiple downstream microservices via HTTP.

### APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| GET/HEAD | `/` | Currency, Product Catalog, Cart, Ad |
| GET/HEAD | `/product/{id}` | Product Catalog, Currency, Cart, Recommendation, Ad |
| GET/HEAD | `/cart` | Currency, Cart, Product Catalog, Recommendation, Shipping |
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

---

## Cart Service Entry Points → Downstream Services

### Summary
A stateful HTTP service that stores and retrieves user cart items. Storage backend can be **in-memory**, **Redis**, or **AlloyDB** depending on environment variables. It does **not** call other microservices.

### APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/cart/{user_id}/items` | None |
| GET | `/cart/{user_id}` | None |
| DELETE | `/cart/{user_id}` | None |
| GET | `/_healthz` | None |

---

## Currency Service Entry Points → Downstream Services

### Summary
An HTTP service that performs currency conversion between ISO 4217 currencies. It loads conversion rates from a local JSON file and converts amounts using EUR as an intermediary base currency. It does **not** call other microservices.

### APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| GET | `/currencies` | None |
| POST | `/convert` | None |
| GET | `/_healthz` | None |

---

## Product Catalog Service Entry Points → Downstream Services

### Summary
An HTTP service that serves product data from an in-memory catalog (loaded/watched from a local products file). Supports listing, fetching by ID, batch fetch, and simple search. It does **not** call other microservices.

### APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| GET | `/products` | None |
| GET | `/products/{id}` | None |
| POST | `/products/batch` | None |
| GET | `/products/search` | None |
| GET | `/_healthz` | None |

---

## Recommendation Service Entry Points → Downstream Services

### Summary
An HTTP service that returns recommended product IDs (or recommendations) based on user/product context. It does **not** call other microservices (frontend may fetch product details separately).

### APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| GET | `/_healthz` | None |
| POST | `/recommendations` | None |

---

## Ad Service Entry Points → Downstream Services

### Summary
An HTTP service that returns contextual ads based on `context_keys` query params. Ads are selected from an in-memory dataset. It does **not** call other microservices.

### APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| GET | `/ads` | None |
| GET | `/_healthz` | None |


---

## Shipping Service Entry Points → Downstream Services

### Summary
An HTTP service that provides shipping quotes and mock shipping/tracking IDs for orders. It does **not** call other microservices.

### APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/shipping/quote` | None |
| POST | `/shipping/ship` | None |
| GET | `/_healthz` | None |

---

## Payment Service Entry Points → Downstream Services

### Summary
An HTTP service that validates credit cards and returns a synthetic transaction ID on successful charge requests. It does **not** call other microservices.

### APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/charge` | None |
| GET | `/_healthz` | None |


---

## Email Service Entry Points → Downstream Services

### Summary
An HTTP service that handles sending order confirmation emails. In dummy mode, it logs the request rather than sending a real email. It does **not** call other microservices.

### APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/send-order-confirmation` | None |
| GET | `/_healthz` | None |


---

## Checkout Service Entry Points → Downstream Services

### Summary
An orchestration HTTP service that processes checkout by:
1) reading the user cart, 2) fetching product prices, 3) converting currencies, 4) quoting + initiating shipping, 5) charging payment, and 6) sending confirmation email.

### APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/placeorder` | Cart, Product Catalog, Currency, Shipping, Payment, Email |
| GET | `/_healthz` | None |

---

## Shopping Assistant Service Entry Points → Downstream Services

### Summary
An HTTP service that accepts a chat prompt (and optional image URL), uses an LLM backend to generate/embedding prompts, performs product search via a DB-backed product store, then returns a final response to the caller. It does **not** call other *boutique microservices* directly, but it may call external systems (DB + LLM provider) depending on configuration.

### APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/` | None (microservices). External: DB backend + LLM backend (config-dependent) |
| GET | `/_healthz` | None |

---
