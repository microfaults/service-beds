# Checkout Service Entry Points → Downstream Services

## Summary
An HTTP service that orchestrates the **end-to-end checkout flow** for a user’s cart.

On `POST /placeorder`, it:
- Fetches the user cart (**Cart Service**)
- Fetches product details (**Product Catalog Service**)
- Converts prices + shipping to the user’s currency (**Currency Service**)
- Requests a shipping quote + creates a shipment (**Shipping Service**)
- Charges the credit card (**Payment Service**)
- Empties the cart (**Cart Service**)
- Sends an order confirmation (**Email Service**)

---

## APIs / Entry Points

| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/placeorder` | Cart, Product Catalog, Currency, Shipping, Payment, Email |
| GET | `/_healthz` | None |

