# Ad Service Entry Points → Downstream Services

## Summary
A HTTP service that returns contextual ads based on `context_keys` query params.  
Ads are selected from an in-memory dataset via `NewService()` / `GetAdsByCategory(...)`.  
This service does **not** call any downstream microservices.
---
## APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| GET | `/ads` | None |
| GET | `/_healthz` | None |