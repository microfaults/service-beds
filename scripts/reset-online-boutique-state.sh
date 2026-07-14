#!/usr/bin/env bash
# Resets online-boutique to a clean state between experiment phases.
# Hand-run between every (baseline -> 1a -> 1b -> 2a -> 2b -> combination)
# transition. Specific to this microservices-demo-go deployment;
# do not generalize.
set -euo pipefail

NS="${NAMESPACE:-default}"

# 0. Load generator must be scaled to zero during phase transitions and
#    experiments (verify-readiness.sh enforces the same invariant).
if kubectl -n "$NS" get deployment loadgenerator >/dev/null 2>&1; then
  kubectl -n "$NS" scale deployment loadgenerator --replicas=0
fi

# 1. Manteion before services: atropos-go >= v0.1.0 fails startup hard when
#    MANTEION_URL is set but unreachable, so the control plane must be up
#    before any service pod restarts. k8s restart backoff covers a brief
#    gap, but do not proceed past a control plane that is not coming up.
if kubectl -n "$NS" get deployment manteion >/dev/null 2>&1; then
  kubectl -n "$NS" rollout status deployment manteion --timeout=120s
fi

# 2. Flush Redis cart store
REDIS_POD=$(kubectl -n "$NS" get pod -l app=redis-cart -o name | head -1)
if [[ -z "$REDIS_POD" ]]; then
  echo "no redis-cart pod found in namespace $NS" >&2
  exit 1
fi
kubectl -n "$NS" exec "$REDIS_POD" -- redis-cli FLUSHALL

# 3. Roll cartservice to drop in-memory fallback state (120s tolerates a
#    crash-loop backoff cycle if manteion only just came up)
kubectl -n "$NS" rollout restart deployment cartservice
kubectl -n "$NS" rollout status deployment cartservice --timeout=120s

# 4. Kafka reset (only if checkoutservice has ENABLE_KAFKA=1)
if kubectl -n "$NS" get deployment kafka >/dev/null 2>&1; then
  kubectl -n "$NS" rollout restart deployment kafka
  kubectl -n "$NS" rollout status deployment kafka --timeout=120s
fi

# 5. Smoke check — every service must answer its liveness route AND report
#    a live manteion connection before the next phase. /atropos/health
#    returns HTTP 200 even when the SDK is offline, so parse the JSON
#    "status" field instead of trusting the HTTP code: connected is
#    required, degraded only warns, disconnected/offline/unparseable (or
#    an HTTP error such as 503) fails. Everything execs through the
#    frontend pod (its image ships wget); targets are the in-cluster
#    Service DNS names and ports from kubernetes-manifests/, except the
#    frontend itself, which is probed on localhost to avoid hairpinning
#    through its own Service VIP.
SERVICES="
adservice|http://adservice:9555
cartservice|http://cartservice:7070
checkoutservice|http://checkoutservice:5050
currencyservice|http://currencyservice:7000
emailservice|http://emailservice:5000
frontend|http://localhost:8080
paymentservice|http://paymentservice:50051
productcatalogservice|http://productcatalogservice:3550
recommendationservice|http://recommendationservice:8080
shippingservice|http://shippingservice:50051
shoppingassistantservice|http://shoppingassistantservice:80
"
SMOKE_FAILS=0
for entry in $SERVICES; do
  svc="${entry%%|*}"
  base="${entry##*|}"

  if ! kubectl -n "$NS" exec deploy/frontend -- wget -q -O /dev/null "${base}/_healthz"; then
    echo "[FAIL] ${svc}: /_healthz not answering" >&2
    SMOKE_FAILS=$((SMOKE_FAILS + 1))
    continue
  fi

  if ! HEALTH=$(kubectl -n "$NS" exec deploy/frontend -- wget -q -O- "${base}/atropos/health"); then
    echo "[FAIL] ${svc}: /atropos/health not answering" >&2
    SMOKE_FAILS=$((SMOKE_FAILS + 1))
    continue
  fi
  STATUS=$(printf '%s' "$HEALTH" | sed -nE 's/.*"status"[[:space:]]*:[[:space:]]*"([^"]*)".*/\1/p')
  case "$STATUS" in
    connected)
      echo "[OK]   ${svc}: healthy, manteion connected" ;;
    degraded)
      echo "[WARN] ${svc}: manteion connection degraded" >&2 ;;
    *)
      echo "[FAIL] ${svc}: /atropos/health status='${STATUS:-unparseable}' (want connected)" >&2
      SMOKE_FAILS=$((SMOKE_FAILS + 1)) ;;
  esac
done

if [[ "$SMOKE_FAILS" -gt 0 ]]; then
  echo "state reset FAILED: ${SMOKE_FAILS} service(s) unhealthy or offline" >&2
  exit 1
fi

echo "state reset complete"
