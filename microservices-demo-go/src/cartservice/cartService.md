# Cart Service Entry Points → Downstream Services

## Summary
A stateful HTTP service that manages a user’s shopping cart (add items, retrieve cart contents, empty cart).

Storage backend is selected at startup:
- **Redis** if `REDIS_ADDR` is set
- **AlloyDB** if `ALLOYDB_PRIMARY_IP` is set (with additional DB env vars)
- Otherwise falls back to an **in-memory** store

The Cart service does **not** call other microservices.

---

## APIs / Entry Points

| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/cart/{user_id}/items` | None |
| GET | `/cart/{user_id}` | None |
| DELETE | `/cart/{user_id}` | None |
| GET | `/_healthz` | None |

