# Recommendation Service Entry Points → Downstream Services

## Summary
An HTTP service that generates product recommendations based on a list of input product IDs.  
It analyzes the input items and returns a list of recommended product IDs.

The service does not call other microservices.  
It performs recommendation logic internally (rule-based or precomputed logic).

---

## APIs / Entry Points

| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/recommendations` | None |
| GET | `/_healthz` | None |