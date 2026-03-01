# Currency Service Entry Points → Downstream Services

## Summary
An HTTP service that performs currency conversion between ISO 4217 currencies.

It:
- Loads conversion rates from a local JSON file
- Converts between currencies using EUR as an intermediary base currency
- Supports listing available currency codes


---

## APIs / Entry Points

| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| GET | `/currencies` | None |
| POST | `/convert` | None |
| GET | `/_healthz` | None |

