# Atropos Instrumentation Across Microservices-Demo-Go

**Date:** 2026-03-10
**Approach:** Middleware-only (Option A)

## Goal

Replace all custom OTel setup with atropos-go's `Init()`, `IngressMiddleware()`, and `EgressTransport()` across all 12 services. Every HTTP endpoint gets an ingress span, every outbound HTTP call gets an egress span, traces propagate end-to-end via W3C TraceContext.

## Per-Service Pattern

```go
import "atropos-go"

func main() {
    ctx := context.Background()
    shutdown, err := atropos.Init(ctx,
        atropos.WithServiceName("<service-name>"),
        atropos.WithServiceVersion("0.1.0"),
    )
    if err != nil { log.Fatal(err) }
    defer shutdown(ctx)

    mux := http.NewServeMux()
    // ... register handlers ...
    handler := atropos.IngressMiddleware(mux, "<service-name>")

    // For services with downstream HTTP calls:
    httpClient := &http.Client{
        Transport: atropos.EgressTransport(http.DefaultTransport),
    }

    http.ListenAndServe(":PORT", handler)
}
```

## Service Matrix

| Service | Ingress | Egress Targets | Remove Existing OTel |
|---|---|---|---|
| AdService | Wrap mux | None | No |
| CartService | Wrap mux | None | No |
| CheckoutService | Wrap mux | Cart, ProductCatalog, Shipping, Currency, Payment, Email | Yes: `telemetry/tracing.go`, otelhttp |
| CurrencyService | Wrap mux | None | Yes: `telemetry/tracing.go`, otelhttp |
| EmailService | Wrap mux | None | No |
| Frontend | Wrap gorilla mux | ProductCatalog, Cart, Currency, Recommendation, Checkout, Shipping, Ad, ShoppingAssistant | Yes: `telemetry/tracing.go`, otelhttp |
| PaymentService | Wrap mux | None | Yes: `telemetry/tracing.go`, otelhttp |
| ProductCatalogService | Wrap mux | None | No |
| RecommendationService | Wrap mux | ProductCatalog | No |
| ShippingService | Wrap mux | None | Yes: custom OTLP setup in main.go |
| ShoppingAssistantService | Wrap mux | None (Gemini uses SDK client) | No |

## Removals

For services with existing OTel (Checkout, Currency, Frontend, Payment, Shipping):
- Delete `telemetry/tracing.go` files
- Remove `otelhttp.NewTransport()` calls → replace with `atropos.EgressTransport()`
- Remove direct `otlptracegrpc` exporter setup
- Remove `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` dependency
- Remove `ENABLE_TRACING` / `DISABLE_TRACING` env var gating

## go.mod Changes

Each service adds:
```
require atropos-go v0.0.0
replace atropos-go => ../../../atropos-go
```

Remove direct OTel SDK/exporter/contrib deps that atropos-go provides transitively.

## Trace Topology

A request through Frontend -> Checkout -> Payment produces:
- Frontend ingress span (incoming request)
- Frontend egress span (call to Checkout)
- Checkout ingress span (incoming from Frontend)
- Checkout egress span (call to Payment)
- Payment ingress span (incoming from Checkout)

All connected via W3C TraceContext propagation set up by `atropos.Init()`.
