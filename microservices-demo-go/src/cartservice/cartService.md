# Cart Service Entry Points → Downstream Services

## Summary
An HTTP service that manages user shopping carts.  
It supports adding items to a cart, retrieving the current cart contents, and emptying a cart.  

---
## APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/cart/{user_id}/items` | None |
| GET | `/cart/{user_id}` | None |
| DELETE | `/cart/{user_id}` | None |
| GET | `/_healthz` | None |