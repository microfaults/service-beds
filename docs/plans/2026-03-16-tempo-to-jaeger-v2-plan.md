# Tempo → Jaeger v2 Migration Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace Grafana Tempo with Jaeger v2 2.16.0 as the trace backend and fix missing `COLLECTOR_SERVICE_ADDR` env vars across services.

**Architecture:** Services export OTLP traces to OTel Collector, which forwards to Jaeger v2 all-in-one (Badger storage). Jaeger serves its own UI and gRPC query API. Grafana remains for metrics only (Prometheus).

**Tech Stack:** Jaeger v2 2.16.0, OTel Collector Contrib 0.100.0, Kubernetes/Kustomize

---

### Task 1: Create Jaeger v2 manifest

**Files:**
- Create: `microservices-demo-go/kubernetes-manifests/jaeger.yaml`

**Step 1: Create the manifest**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: jaeger-conf
data:
  jaeger.yaml: |
    extensions:
      jaeger_storage:
        badger:
          directories:
            values: /var/jaeger/traces
            keys: /var/jaeger/keys
          ephemeral: false
      jaeger_query:
        storage:
          traces: badger
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
    exporters:
      jaeger_storage_exporter:
        trace_storage: badger
    service:
      extensions: [jaeger_storage, jaeger_query]
      pipelines:
        traces:
          receivers: [otlp]
          exporters: [jaeger_storage_exporter]
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jaeger
  template:
    metadata:
      labels:
        app: jaeger
    spec:
      containers:
        - name: jaeger
          image: jaegertracing/jaeger:2.16.0
          args:
            - "--config=/conf/jaeger.yaml"
          ports:
            - containerPort: 4317
            - containerPort: 16686
            - containerPort: 16685
          resources:
            requests:
              cpu: 200m
              memory: 256Mi
            limits:
              cpu: 300m
              memory: 512Mi
          volumeMounts:
            - name: jaeger-config
              mountPath: /conf
            - name: jaeger-data
              mountPath: /var/jaeger
      volumes:
        - name: jaeger-config
          configMap:
            name: jaeger-conf
        - name: jaeger-data
          emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: jaeger
spec:
  ports:
    - name: otlp-grpc
      port: 4317
      targetPort: 4317
    - name: ui
      port: 16686
      targetPort: 16686
    - name: grpc-query
      port: 16685
      targetPort: 16685
  selector:
    app: jaeger
```

**Step 2: Verify YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load_all(open('microservices-demo-go/kubernetes-manifests/jaeger.yaml')); print('OK')"`
Expected: `OK`

**Step 3: Commit**

```bash
git add microservices-demo-go/kubernetes-manifests/jaeger.yaml
git commit -m "feat: add Jaeger v2 manifest (replaces Tempo)"
```

---

### Task 2: Remove Tempo manifest

**Files:**
- Delete: `microservices-demo-go/kubernetes-manifests/tempo.yaml`

**Step 1: Delete the file**

```bash
git rm microservices-demo-go/kubernetes-manifests/tempo.yaml
```

**Step 2: Commit**

```bash
git commit -m "chore: remove Tempo manifest"
```

---

### Task 3: Update OTel Collector exporter

**Files:**
- Modify: `microservices-demo-go/kubernetes-manifests/otel-collector.yaml:12-16,25`

**Step 1: Rename exporter from `otlp/tempo` to `otlp/jaeger` and update endpoint**

Replace lines 12-16:
```yaml
      otlp/tempo:
        endpoint: "tempo:4317"
        tls:
          insecure: true
```
With:
```yaml
      otlp/jaeger:
        endpoint: "jaeger:4317"
        tls:
          insecure: true
```

Replace line 25:
```yaml
          exporters: [spanmetrics, otlp/tempo]
```
With:
```yaml
          exporters: [spanmetrics, otlp/jaeger]
```

**Step 2: Verify YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load_all(open('microservices-demo-go/kubernetes-manifests/otel-collector.yaml')); print('OK')"`
Expected: `OK`

**Step 3: Commit**

```bash
git add microservices-demo-go/kubernetes-manifests/otel-collector.yaml
git commit -m "feat: point OTel Collector trace exporter to Jaeger"
```

---

### Task 4: Remove Tempo datasource from Grafana

**Files:**
- Modify: `microservices-demo-go/kubernetes-manifests/grafana.yaml:14-24`

**Step 1: Remove the Tempo datasource block**

Remove lines 14-24 (the entire `- name: Tempo` block including jsonData):
```yaml
      - name: Tempo
        type: tempo
        access: proxy
        url: http://tempo:3200
        jsonData:
          tracesToMetrics:
            datasourceUid: prometheus
          nodeGraph:
            enabled: true
          serviceMap:
            datasourceUid: prometheus
```

**Step 2: Verify YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load_all(open('microservices-demo-go/kubernetes-manifests/grafana.yaml')); print('OK')"`
Expected: `OK`

**Step 3: Commit**

```bash
git add microservices-demo-go/kubernetes-manifests/grafana.yaml
git commit -m "chore: remove Tempo datasource from Grafana"
```

---

### Task 5: Update kustomization.yaml

**Files:**
- Modify: `microservices-demo-go/kubernetes-manifests/kustomization.yaml:30`

**Step 1: Replace tempo.yaml with jaeger.yaml**

Replace line 30:
```yaml
 - tempo.yaml
```
With:
```yaml
 - jaeger.yaml
```

**Step 2: Commit**

```bash
git add microservices-demo-go/kubernetes-manifests/kustomization.yaml
git commit -m "chore: replace tempo with jaeger in kustomization"
```

---

### Task 6: Add COLLECTOR_SERVICE_ADDR to services missing it

**Files:**
- Modify: `microservices-demo-go/kubernetes-manifests/adservice.yaml:51`
- Modify: `microservices-demo-go/kubernetes-manifests/cartservice.yaml:51`
- Modify: `microservices-demo-go/kubernetes-manifests/checkoutservice.yaml:72`
- Modify: `microservices-demo-go/kubernetes-manifests/emailservice.yaml:53`
- Modify: `microservices-demo-go/kubernetes-manifests/productcatalogservice.yaml:53`
- Modify: `microservices-demo-go/kubernetes-manifests/recommendationservice.yaml:65`
- Modify: `microservices-demo-go/kubernetes-manifests/shippingservice.yaml:52`

**Step 1: Add env var to each manifest**

After the last env var in each file, add:
```yaml
        - name: COLLECTOR_SERVICE_ADDR
          value: "otel-collector:4317"
```

Note: `checkoutservice.yaml` uses 10-space indentation for env vars. All others use 8-space.

**Step 2: Verify all 7 files parse correctly**

Run:
```bash
for f in adservice cartservice checkoutservice emailservice productcatalogservice recommendationservice shippingservice; do
  python3 -c "import yaml; list(yaml.safe_load_all(open('microservices-demo-go/kubernetes-manifests/${f}.yaml'))); print('${f}: OK')"
done
```
Expected: all 7 print OK

**Step 3: Verify all 10 services now have COLLECTOR_SERVICE_ADDR**

Run: `grep -rl COLLECTOR_SERVICE_ADDR microservices-demo-go/kubernetes-manifests/ | wc -l`
Expected: `10` (7 newly added + frontend + paymentservice + currencyservice)

**Step 4: Commit**

```bash
git add microservices-demo-go/kubernetes-manifests/{adservice,cartservice,checkoutservice,emailservice,productcatalogservice,recommendationservice,shippingservice}.yaml
git commit -m "fix: add COLLECTOR_SERVICE_ADDR to all service manifests"
```

---

### Task 7: Smoke test kustomize build

**Step 1: Verify kustomize renders without errors**

Run: `kubectl kustomize microservices-demo-go/kubernetes-manifests/`
Expected: valid multi-document YAML output with jaeger resources and no tempo references

**Step 2: Verify no tempo references remain**

Run: `grep -ri tempo microservices-demo-go/kubernetes-manifests/`
Expected: no output

**Step 3: Verify jaeger resources present**

Run: `kubectl kustomize microservices-demo-go/kubernetes-manifests/ | grep -c 'name: jaeger'`
Expected: `3` or more (ConfigMap, Deployment, Service)
