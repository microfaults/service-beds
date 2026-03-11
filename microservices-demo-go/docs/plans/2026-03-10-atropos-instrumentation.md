# Atropos Instrumentation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace all custom OTel setup with atropos-go middleware-only instrumentation across all 12 microservices, giving every HTTP endpoint an ingress span and every outbound HTTP call an egress span.

**Architecture:** Each service calls `atropos.Init()` at startup, wraps its HTTP server handler with `atropos.IngressMiddleware()`, and wraps outbound HTTP client transports with `atropos.EgressTransport()`. Services that had custom `telemetry/tracing.go` files and `otelhttp` usage get those removed entirely.

**Tech Stack:** atropos-go (module `atropos-go`), Go standard library `net/http`, gorilla/mux (Frontend only)

---

## Group 1: Simple Services (no existing OTel, no egress)

These services have no tracing today and make no downstream HTTP calls. Each needs: go.mod update, atropos.Init(), IngressMiddleware wrapping the mux.

### Task 1: AdService — add atropos instrumentation

**Files:**
- Modify: `src/adservice/go.mod`
- Modify: `src/adservice/main.go`

**Step 1: Update go.mod**

Add atropos-go dependency and replace directive. In `src/adservice/go.mod`:

```
module github.com/microservices-demo-go/adservice

go 1.25.5

require atropos-go v0.0.0

replace atropos-go => ../../../../atropos-go
```

**Step 2: Run go mod tidy**

Run: `cd src/adservice && go mod tidy`
Expected: go.sum updated, indirect deps resolved

**Step 3: Modify main.go**

AdService uses `http.HandleFunc()` (DefaultServeMux) with `http.Server{Handler: nil}`. Convert to explicit mux and wrap with atropos.

Add to imports:
```go
"atropos-go"
```

Add atropos.Init() at the top of main(), before handler registration:
```go
shutdown, err := atropos.Init(ctx,
    atropos.WithServiceName("adservice"),
    atropos.WithServiceVersion("0.1.0"),
)
if err != nil {
    logger.Error("failed to init atropos", "error", err)
    os.Exit(1)
}
defer shutdown(ctx)
```

Convert DefaultServeMux handlers to explicit mux:
```go
mux := http.NewServeMux()
mux.HandleFunc("POST /ads", func(w http.ResponseWriter, r *http.Request) { ... })
mux.HandleFunc("GET /_healthz", func(w http.ResponseWriter, r *http.Request) { ... })
```

Wrap mux with IngressMiddleware and set as server handler:
```go
srv := &http.Server{
    Addr:    ":" + port,
    Handler: atropos.IngressMiddleware(mux, "adservice"),
}
```

**Step 4: Verify it compiles**

Run: `cd src/adservice && go build ./...`
Expected: Builds successfully

**Step 5: Commit**

```bash
git add src/adservice/
git commit -m "feat(adservice): add atropos ingress instrumentation"
```

---

### Task 2: CartService — add atropos instrumentation

**Files:**
- Modify: `src/cartservice/src/go.mod`
- Modify: `src/cartservice/src/cmd/cartservice/main.go`

**Step 1: Update go.mod**

In `src/cartservice/src/go.mod`, add:
```
require atropos-go v0.0.0

replace atropos-go => ../../../../../atropos-go
```

**Step 2: Run go mod tidy**

Run: `cd src/cartservice/src && go mod tidy`

**Step 3: Modify main.go**

CartService already uses `mux := http.NewServeMux()` and `http.ListenAndServe(fmt.Sprintf(":%s", port), mux)`.

Add to imports:
```go
"atropos-go"
"context"
```

Note: `context` may already be imported — check first.

Add atropos.Init() near top of main():
```go
ctx := context.Background()
shutdown, err := atropos.Init(ctx,
    atropos.WithServiceName("cartservice"),
    atropos.WithServiceVersion("0.1.0"),
)
if err != nil {
    log.Fatalf("failed to init atropos: %v", err)
}
defer shutdown(ctx)
```

Wrap the mux before passing to ListenAndServe:
```go
if err := http.ListenAndServe(fmt.Sprintf(":%s", port), atropos.IngressMiddleware(mux, "cartservice")); err != nil {
    log.Fatalf("failed to serve HTTP: %v", err)
}
```

**Step 4: Verify it compiles**

Run: `cd src/cartservice/src && go build ./...`

**Step 5: Commit**

```bash
git add src/cartservice/
git commit -m "feat(cartservice): add atropos ingress instrumentation"
```

---

### Task 3: EmailService — add atropos instrumentation

**Files:**
- Modify: `src/emailservice/go.mod`
- Modify: `src/emailservice/main.go`

**Step 1: Update go.mod**

In `src/emailservice/go.mod`, add:
```
require atropos-go v0.0.0

replace atropos-go => ../../../../atropos-go
```

**Step 2: Run go mod tidy**

Run: `cd src/emailservice && go mod tidy`

**Step 3: Modify main.go**

EmailService uses `http.HandleFunc()` (DefaultServeMux) and `http.ListenAndServe(addr, nil)`. Convert to explicit mux.

Add to imports:
```go
"atropos-go"
"context"
```

Add atropos.Init() near top of main():
```go
ctx := context.Background()
shutdown, err := atropos.Init(ctx,
    atropos.WithServiceName("emailservice"),
    atropos.WithServiceVersion("0.1.0"),
)
if err != nil {
    logger.Error(fmt.Sprintf("failed to init atropos: %s", err.Error()))
    os.Exit(1)
}
defer shutdown(ctx)
```

Convert to explicit mux:
```go
mux := http.NewServeMux()
mux.HandleFunc("/send-order-confirmation", handleSendOrderConfirmation)
mux.HandleFunc("/_healthz", handleHealthCheck)
```

Replace `http.ListenAndServe(addr, nil)` with:
```go
if err := http.ListenAndServe(addr, atropos.IngressMiddleware(mux, "emailservice")); err != nil {
    logger.Error(fmt.Sprintf("Server failed: %s", err.Error()))
}
```

**Step 4: Verify it compiles**

Run: `cd src/emailservice && go build ./...`

**Step 5: Commit**

```bash
git add src/emailservice/
git commit -m "feat(emailservice): add atropos ingress instrumentation"
```

---

### Task 4: ProductCatalogService — add atropos instrumentation

**Files:**
- Modify: `src/productcatalogservice/go.mod`
- Modify: `src/productcatalogservice/main.go`

**Step 1: Update go.mod**

In `src/productcatalogservice/go.mod`, add:
```
require atropos-go v0.0.0

replace atropos-go => ../../../../atropos-go
```

**Step 2: Run go mod tidy**

Run: `cd src/productcatalogservice && go mod tidy`

**Step 3: Modify main.go**

ProductCatalogService already uses `mux := http.NewServeMux()` and `http.ListenAndServe(":"+port, mux)`.

Add to imports:
```go
"atropos-go"
```

Note: `context` is already imported.

Add atropos.Init() near top of main():
```go
shutdown, err := atropos.Init(ctx,
    atropos.WithServiceName("productcatalogservice"),
    atropos.WithServiceVersion("0.1.0"),
)
if err != nil {
    log.Fatal(err)
}
defer shutdown(ctx)
```

Note: `ctx` is already created for the file watcher. Place Init after ctx creation.

Wrap mux:
```go
if err := http.ListenAndServe(":"+port, atropos.IngressMiddleware(mux, "productcatalogservice")); err != nil {
    log.Fatal(err)
}
```

**Step 4: Verify it compiles**

Run: `cd src/productcatalogservice && go build ./...`

**Step 5: Commit**

```bash
git add src/productcatalogservice/
git commit -m "feat(productcatalogservice): add atropos ingress instrumentation"
```

---

### Task 5: ShoppingAssistantService — add atropos instrumentation

**Files:**
- Modify: `src/shoppingassistantservice/go.mod`
- Modify: `src/shoppingassistantservice/main.go`

**Step 1: Update go.mod**

In `src/shoppingassistantservice/go.mod`, add:
```
require atropos-go v0.0.0

replace atropos-go => ../../../../atropos-go
```

**Step 2: Run go mod tidy**

Run: `cd src/shoppingassistantservice && go mod tidy`

**Step 3: Modify main.go**

ShoppingAssistantService uses `http.HandleFunc()` (DefaultServeMux) and `http.ListenAndServe(":"+port, nil)`. Convert to explicit mux.

Add to imports:
```go
"atropos-go"
```

Note: `context` is already imported.

Add atropos.Init() near top of main() (after ctx creation):
```go
shutdown, err := atropos.Init(ctx,
    atropos.WithServiceName("shoppingassistantservice"),
    atropos.WithServiceVersion("0.1.0"),
)
if err != nil {
    log.Fatalf("failed to init atropos: %v", err)
}
defer shutdown(ctx)
```

Convert to explicit mux:
```go
mux := http.NewServeMux()
mux.HandleFunc("/", handleAssistant)
mux.HandleFunc("/_healthz", handleHealth)
```

Replace `http.ListenAndServe(":"+port, nil)` with:
```go
if err := http.ListenAndServe(":"+port, atropos.IngressMiddleware(mux, "shoppingassistantservice")); err != nil {
    log.Fatalf("failed to start server: %v", err)
}
```

**Step 4: Verify it compiles**

Run: `cd src/shoppingassistantservice && go build ./...`

**Step 5: Commit**

```bash
git add src/shoppingassistantservice/
git commit -m "feat(shoppingassistantservice): add atropos ingress instrumentation"
```

---

## Group 2: Simple Service with Egress

### Task 6: RecommendationService — add atropos ingress + egress

**Files:**
- Modify: `src/recommendationservice/go.mod`
- Modify: `src/recommendationservice/main.go`

**Step 1: Update go.mod**

In `src/recommendationservice/go.mod`, add:
```
require atropos-go v0.0.0

replace atropos-go => ../../../../atropos-go
```

**Step 2: Run go mod tidy**

Run: `cd src/recommendationservice && go mod tidy`

**Step 3: Modify main.go**

RecommendationService uses DefaultServeMux and has an HTTP client `&http.Client{Timeout: 10 * time.Second}` for calling ProductCatalogService.

Add to imports:
```go
"atropos-go"
"context"
```

Add atropos.Init() near top of main():
```go
ctx := context.Background()
shutdown, err := atropos.Init(ctx,
    atropos.WithServiceName("recommendationservice"),
    atropos.WithServiceVersion("0.1.0"),
)
if err != nil {
    log.Fatalf("failed to init atropos: %v", err)
}
defer shutdown(ctx)
```

Wrap the HTTP client transport (where the service struct is created):
```go
svc := &RecommendationService{
    Client: &http.Client{
        Transport: atropos.EgressTransport(http.DefaultTransport),
        Timeout:   10 * time.Second,
    },
    CatalogAddr: catalogAddr,
}
```

Convert to explicit mux and wrap with IngressMiddleware:
```go
mux := http.NewServeMux()
mux.HandleFunc("/recommendations", svc.handleRecommendations)
mux.HandleFunc("/_healthz", handleHealth)

if err := http.ListenAndServe(":"+port, atropos.IngressMiddleware(mux, "recommendationservice")); err != nil {
    log.Fatalf("failed to serve: %v", err)
}
```

**Step 4: Verify it compiles**

Run: `cd src/recommendationservice && go build ./...`

**Step 5: Commit**

```bash
git add src/recommendationservice/
git commit -m "feat(recommendationservice): add atropos ingress + egress instrumentation"
```

---

## Group 3: Replace Existing OTel (no egress)

These services have `telemetry/tracing.go`, `otelhttp` usage, and `ENABLE_TRACING`/`DISABLE_TRACING` env var checks. Replace all of that with atropos.

### Task 7: CurrencyService — replace OTel with atropos

**Files:**
- Modify: `src/currencyservice/go.mod`
- Modify: `src/currencyservice/main.go`
- Delete: `src/currencyservice/telemetry/tracing.go` (and `telemetry/` directory)

**Step 1: Delete telemetry package**

Run: `rm -rf src/currencyservice/telemetry/`

**Step 2: Update go.mod**

In `src/currencyservice/go.mod`:
- Add: `require atropos-go v0.0.0`
- Add: `replace atropos-go => ../../../../atropos-go`
- The direct OTel deps (`otelhttp`, `otel`, `otlptracegrpc`, `otel/sdk`) will be cleaned up by go mod tidy since atropos-go provides them transitively.

**Step 3: Modify main.go**

Remove from imports:
```go
"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
telemetry "github.com/GoogleCloudPlatform/microservices-demo/src/currencyservice/telemetry"
```

Add to imports:
```go
"atropos-go"
```

Replace the ENABLE_TRACING block (lines ~70-78):
```go
if os.Getenv("ENABLE_TRACING") == "1" {
    log.Info("Tracing enabled.")
    tp, err := telemetry.InitTracing(log, ctx)
    ...
}
```

With:
```go
shutdown, err := atropos.Init(ctx,
    atropos.WithServiceName("currencyservice"),
    atropos.WithServiceVersion("0.1.0"),
)
if err != nil {
    log.Warnf("failed to init atropos: %v", err)
}
if shutdown != nil {
    defer shutdown(ctx)
}
```

Replace the otelhttp handler wrapping (lines ~106-107):
```go
var handler http.Handler = mux
handler = otelhttp.NewHandler(handler, "currencyservice")
```

With:
```go
var handler http.Handler = atropos.IngressMiddleware(mux, "currencyservice")
```

Update ListenAndServe to use `handler`.

**Step 4: Run go mod tidy**

Run: `cd src/currencyservice && go mod tidy`

**Step 5: Verify it compiles**

Run: `cd src/currencyservice && go build ./...`

**Step 6: Commit**

```bash
git add src/currencyservice/
git commit -m "feat(currencyservice): replace custom OTel with atropos instrumentation"
```

---

### Task 8: PaymentService — replace OTel with atropos

**Files:**
- Modify: `src/paymentservice/go.mod`
- Modify: `src/paymentservice/main.go`
- Delete: `src/paymentservice/telemetry/tracing.go` (and `telemetry/` directory)

**Step 1: Delete telemetry package**

Run: `rm -rf src/paymentservice/telemetry/`

**Step 2: Update go.mod**

In `src/paymentservice/go.mod`:
- Add: `require atropos-go v0.0.0`
- Add: `replace atropos-go => ../../../../atropos-go`

**Step 3: Modify main.go**

Remove from imports:
```go
"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
telemetry "github.com/GoogleCloudPlatform/microservices-demo/src/paymentservice/telemetry"
```

Add to imports:
```go
"atropos-go"
```

Replace the ENABLE_TRACING block (lines ~74-82) with:
```go
shutdown, err := atropos.Init(ctx,
    atropos.WithServiceName("paymentservice"),
    atropos.WithServiceVersion("0.1.0"),
)
if err != nil {
    log.Warnf("failed to init atropos: %v", err)
}
if shutdown != nil {
    defer shutdown(ctx)
}
```

Replace the otelhttp handler wrapping (lines ~105-106):
```go
var handler http.Handler = mux
handler = otelhttp.NewHandler(handler, "paymentservice")
```

With:
```go
var handler http.Handler = atropos.IngressMiddleware(mux, "paymentservice")
```

**Step 4: Run go mod tidy**

Run: `cd src/paymentservice && go mod tidy`

**Step 5: Verify it compiles**

Run: `cd src/paymentservice && go build ./...`

**Step 6: Commit**

```bash
git add src/paymentservice/
git commit -m "feat(paymentservice): replace custom OTel with atropos instrumentation"
```

---

### Task 9: ShippingService — replace OTel with atropos

**Files:**
- Modify: `src/shippingservice/go.mod`
- Modify: `src/shippingservice/main.go`

**Step 1: Update go.mod**

In `src/shippingservice/go.mod`:
- Add: `require atropos-go v0.0.0`
- Add: `replace atropos-go => ../../../../atropos-go`

**Step 2: Modify main.go**

ShippingService has no separate telemetry/ package — it has inline `initTracer()` and per-handler `otelhttp.NewHandler()` wrapping.

Remove from imports:
```go
"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
"go.opentelemetry.io/otel"
"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
"go.opentelemetry.io/otel/sdk/resource"
sdktrace "go.opentelemetry.io/otel/sdk/trace"
semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
```

Add to imports:
```go
"atropos-go"
```

Replace the DISABLE_TRACING block and initTracer call (lines ~58-64) with:
```go
shutdown, err := atropos.Init(ctx,
    atropos.WithServiceName("shippingservice"),
    atropos.WithServiceVersion("0.1.0"),
)
if err != nil {
    log.Warnf("failed to init atropos: %v", err)
}
if shutdown != nil {
    defer shutdown(ctx)
}
```

Replace per-handler otelhttp wrapping (lines ~72-77):
```go
mux.Handle("POST /shipping/quote", otelhttp.NewHandler(http.HandlerFunc(handleGetQuote), "GetQuote"))
mux.Handle("POST /shipping/ship", otelhttp.NewHandler(http.HandlerFunc(handleShipOrder), "ShipOrder"))
mux.Handle("GET /_healthz", otelhttp.NewHandler(http.HandlerFunc(handleHealth), "HealthCheck"))
```

With plain handler registration:
```go
mux.HandleFunc("POST /shipping/quote", handleGetQuote)
mux.HandleFunc("POST /shipping/ship", handleShipOrder)
mux.HandleFunc("GET /_healthz", handleHealth)
```

Wrap the mux with IngressMiddleware in the http.Server:
```go
srv := &http.Server{
    Addr:    addr,
    Handler: atropos.IngressMiddleware(mux, "shippingservice"),
}
```

Delete the `initTracer()` function entirely (lines ~105-130).

**Step 3: Run go mod tidy**

Run: `cd src/shippingservice && go mod tidy`

**Step 4: Verify it compiles**

Run: `cd src/shippingservice && go build ./...`

**Step 5: Commit**

```bash
git add src/shippingservice/
git commit -m "feat(shippingservice): replace custom OTel with atropos instrumentation"
```

---

## Group 4: Replace Existing OTel + Egress

### Task 10: CheckoutService — replace OTel with atropos (ingress + egress)

**Files:**
- Modify: `src/checkoutService/go.mod`
- Modify: `src/checkoutService/main.go`
- Delete: `src/checkoutService/telemetry/tracing.go` (and `telemetry/` directory)

**Step 1: Delete telemetry package**

Run: `rm -rf src/checkoutService/telemetry/`

**Step 2: Update go.mod**

In `src/checkoutService/go.mod`:
- Add: `require atropos-go v0.0.0`
- Add: `replace atropos-go => ../../../../atropos-go`

**Step 3: Modify main.go**

Remove from imports:
```go
"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
telemetry "checkoutservice/telemetry"
```

Add to imports:
```go
"atropos-go"
```

Replace the ENABLE_TRACING block (lines ~64-72) with:
```go
shutdown, err := atropos.Init(ctx,
    atropos.WithServiceName("checkoutservice"),
    atropos.WithServiceVersion("0.1.0"),
)
if err != nil {
    log.Warnf("failed to init atropos: %v", err)
}
if shutdown != nil {
    defer shutdown(ctx)
}
```

Replace the HTTP client transport (lines ~82-85):
```go
httpClient: &http.Client{
    Transport: otelhttp.NewTransport(http.DefaultTransport),
    Timeout:   10 * time.Second,
},
```

With:
```go
httpClient: &http.Client{
    Transport: atropos.EgressTransport(http.DefaultTransport),
    Timeout:   10 * time.Second,
},
```

Replace the handler wrapping (lines ~118-119):
```go
var handler http.Handler = mux
handler = otelhttp.NewHandler(handler, "checkoutservice")
```

With:
```go
handler := atropos.IngressMiddleware(mux, "checkoutservice")
```

Update ListenAndServe to use `handler`.

**Step 4: Run go mod tidy**

Run: `cd src/checkoutService && go mod tidy`

**Step 5: Verify it compiles**

Run: `cd src/checkoutService && go build ./...`

**Step 6: Commit**

```bash
git add src/checkoutService/
git commit -m "feat(checkoutservice): replace custom OTel with atropos ingress + egress"
```

---

### Task 11: Frontend — replace OTel with atropos (ingress + egress)

**Files:**
- Modify: `src/frontend/go.mod`
- Modify: `src/frontend/main.go`
- Modify: `src/frontend/clients/clients.go`
- Delete: `src/frontend/telemetry/tracing.go` (and `telemetry/` directory)

**Step 1: Delete telemetry package**

Run: `rm -rf src/frontend/telemetry/`

**Step 2: Update go.mod**

In `src/frontend/go.mod`:
- Add: `require atropos-go v0.0.0`
- Add: `replace atropos-go => ../../../../atropos-go`

**Step 3: Modify clients/clients.go — replace otelhttp with atropos egress**

Remove from imports:
```go
"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
```

Add to imports:
```go
"atropos-go"
```

Replace ALL occurrences of:
```go
client: &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
```

With:
```go
client: &http.Client{Transport: atropos.EgressTransport(http.DefaultTransport)},
```

This occurs in 7 client constructors: NewCurrencyClient, NewProductCatalogClient, NewCartClient, NewRecommendationClient, NewShippingClient, NewCheckoutClient, NewAdClient.

**Step 4: Modify main.go — replace init and handler wrapping**

Remove from imports:
```go
"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
telemetry "github.com/GoogleCloudPlatform/microservices-demo-go/src/frontend/telemetry"
```

Add to imports:
```go
"atropos-go"
```

Replace the ENABLE_TRACING block (lines ~72-80) with:
```go
shutdown, err := atropos.Init(ctx,
    atropos.WithServiceName("frontend"),
    atropos.WithServiceVersion("0.1.0"),
)
if err != nil {
    log.Warnf("failed to init atropos: %v", err)
}
if shutdown != nil {
    defer shutdown(ctx)
}
```

Replace the handler middleware stack (lines ~140-143):
```go
var handler http.Handler = r
handler = &logHandler{log: log, next: handler}
handler = ensureSessionID(handler)
handler = otelhttp.NewHandler(handler, "frontend")
```

With (preserving existing middleware, replacing otelhttp with atropos):
```go
var handler http.Handler = r
handler = &logHandler{log: log, next: handler}
handler = ensureSessionID(handler)
handler = atropos.IngressMiddleware(handler, "frontend")
```

Note: `atropos.IngressMiddleware` accepts `http.Handler`, not just mux, so it composes fine with the existing middleware chain.

**Step 5: Run go mod tidy**

Run: `cd src/frontend && go mod tidy`

**Step 6: Verify it compiles**

Run: `cd src/frontend && go build ./...`

**Step 7: Commit**

```bash
git add src/frontend/
git commit -m "feat(frontend): replace custom OTel with atropos ingress + egress"
```

---

## Group 5: Final Verification

### Task 12: Build all services

**Step 1: Build each service**

Run from repo root:
```bash
for svc in adservice emailservice productcatalogservice recommendationservice shippingservice shoppingassistantservice currencyservice paymentservice; do
  echo "=== Building $svc ===" && (cd src/$svc && go build ./...) || echo "FAIL: $svc"
done

echo "=== Building cartservice ===" && (cd src/cartservice/src && go build ./...) || echo "FAIL: cartservice"
echo "=== Building checkoutService ===" && (cd src/checkoutService && go build ./...) || echo "FAIL: checkoutService"
echo "=== Building frontend ===" && (cd src/frontend && go build ./...) || echo "FAIL: frontend"
```

Expected: All build successfully

**Step 2: Final commit if any fixups needed**

```bash
git add -A
git commit -m "fix: resolve any build issues from atropos instrumentation"
```
