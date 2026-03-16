# Tempo → Jaeger v2 Migration Design

## Decision

Replace Grafana Tempo 2.6.1 with Jaeger v2 2.16.0 (all-in-one, Badger storage) as the trace backend. Use Jaeger's native UI for trace analysis instead of Grafana's Tempo datasource. Keep the OTel Collector as a separate component.

## Motivation

- Jaeger v2's gRPC QueryService API provides programmatic trace access for the oracle service (replay attacks from historical requests)
- Jaeger's native UI has trace comparison, dependency DAGs, and deep-linking — purpose-built for trace analysis
- Jaeger v2's SPM partially overlaps spanmetrics, giving native RED metrics
- Tempo's HTTP API is less mature for programmatic use

## Architecture

```
Services (atropos-go SDK, OTLP gRPC :4317)
    → OTel Collector (receivers, spanmetrics connector)
        → Jaeger v2 all-in-one (OTLP :4317, Badger storage)
            → Jaeger UI (:16686)
            → Jaeger gRPC QueryService (:16685) ← oracle service
        → Prometheus exporter (:8889) ← Grafana
```

## Changes

1. **Remove** `tempo.yaml` manifest
2. **Add** `jaeger.yaml` manifest
   - Image: `jaegertracing/jaeger:2.16.0`
   - Badger storage (ephemeral via emptyDir)
   - OTLP gRPC receiver on :4317
   - UI on :16686, gRPC query on :16685
   - Resources: 200m/300m CPU, 256Mi/512Mi memory
3. **Update** OTel Collector config: `otlp/tempo` exporter → `otlp/jaeger` pointing at `jaeger:4317`
4. **Update** Grafana datasource provisioning: remove Tempo datasource
5. **Update** `kustomization.yaml`: replace `tempo.yaml` → `jaeger.yaml`
6. **Fix** `COLLECTOR_SERVICE_ADDR` env var — add to 8 service manifests missing it:
   adservice, cartservice, checkoutservice, emailservice,
   productcatalogservice, recommendationservice, shippingservice, shoppingassistantservice

## No Changes

- Service source code (atropos-go SDK unchanged)
- Grafana dashboards (Prometheus-only queries)
- Prometheus scrape config
- OTel Collector receivers, spanmetrics connector, prometheus exporter

## Instrumentation Concerns (Deferred)

- **AlwaysSample** is the default — will need ratio-based sampling under load
- **No batch/queue limits** on OTel Collector exporters — unbounded memory under burst
- **Badger on emptyDir** — pod restart loses traces; PVC needed if oracle requires cross-run history
