#!/usr/bin/env bash
# Resets online-boutique to a clean state between experiment phases.
# Hand-run between every (baseline -> 1a -> 1b -> 2a -> 2b -> combination)
# transition. Specific to this microservices-demo-go deployment;
# do not generalize.
set -euo pipefail

NS="${NAMESPACE:-default}"

# 1. Flush Redis cart store
REDIS_POD=$(kubectl -n "$NS" get pod -l app=redis-cart -o name | head -1)
if [[ -z "$REDIS_POD" ]]; then
  echo "no redis-cart pod found in namespace $NS" >&2
  exit 1
fi
kubectl -n "$NS" exec "$REDIS_POD" -- redis-cli FLUSHALL

# 2. Roll cartservice to drop in-memory fallback state
kubectl -n "$NS" rollout restart deployment cartservice
kubectl -n "$NS" rollout status deployment cartservice --timeout=60s

# 3. Kafka reset (only if checkoutservice has ENABLE_KAFKA=1)
if kubectl -n "$NS" get deployment kafka >/dev/null 2>&1; then
  kubectl -n "$NS" rollout restart deployment kafka
  kubectl -n "$NS" rollout status deployment kafka --timeout=120s
fi

# 4. Smoke check — frontend must be reachable before next phase
kubectl -n "$NS" exec deploy/frontend -- wget -q -O- localhost:8080/healthz \
  || (echo "frontend smoke failed" >&2 && exit 1)

echo "state reset complete"
