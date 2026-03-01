# Email Service Entry Points → Downstream Services

## Summary
An HTTP service responsible for handling order confirmation email requests.

In its current implementation (dummy mode), the service:
- Receives order confirmation requests
- Logs the email and order details
- Optionally renders a confirmation template
- Returns an empty JSON response

It does not send real emails and does not call other microservices.

---

## APIs / Entry Points

| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/send-order-confirmation` | None |
| GET | `/_healthz` | None |

