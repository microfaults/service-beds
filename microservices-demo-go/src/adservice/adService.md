# Ad Service Entry Points → Downstream Services

## Summary
A HTTP service that serves contextually relevant advertisements based on category keywords. It selects ads from a pre-populated in-memory map, falling back to random ads when no matching categories are found.
---
## APIs / Entry Points
| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| GET | `/ads` | None |
| GET | `/_healthz` | None |