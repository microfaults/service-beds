# Shipping Service Entry Points → Downstream Services

## Summary
An HTTP service that provides shipping quotes and simulates shipment processing.  
It calculates shipping costs and generates tracking IDs for orders.


---

## APIs / Entry Points

| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/shipping/quote` | None |
| POST | `/shipping/ship` | None |
| GET | `/_healthz` | None |