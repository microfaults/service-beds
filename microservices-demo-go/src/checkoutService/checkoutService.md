# Checkout Service Entry Points → Downstream Services

| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/placeorder` | Cart, Product Catalog, Currency, Shipping, Payment, Email |
| GET/POST* | `/_healthz` | None |