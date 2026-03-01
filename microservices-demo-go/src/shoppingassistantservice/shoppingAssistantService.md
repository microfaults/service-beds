# Shopping Assistant Service Entry Points → Downstream Services

## Summary
An HTTP service that provides AI-powered product recommendations.  
It:

1. Uses an LLM to describe a room or user request.
2. Generates embeddings for semantic search.
3. Queries a product database (AlloyDB, Postgres, or mock store).
4. Uses the LLM again to generate final recommendations.

This service integrates AI and vector search capabilities.

---

## APIs / Entry Points

| Method | Path | Downstream Services Called |
|--------|------|----------------------------|
| POST | `/` | LLM Backend, Product Store (DB) |
| GET | `/_healthz` | None |