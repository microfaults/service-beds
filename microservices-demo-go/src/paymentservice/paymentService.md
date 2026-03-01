# Payment Service Entry Points → Downstream Services

## Summary
An HTTP service responsible for validating and processing credit card payments.

It performs:
- Expiration date validation
- Card type validation (Visa / Mastercard)
- Luhn algorithm verification
- Transaction ID generation

---

## APIs / Entry Points

| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/charge` | None |
| GET | `/_healthz` | None |

---

