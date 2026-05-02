#!/usr/bin/env bash
# Pre-experiment readiness check for cache-box isolation experiments.
# Non-destructive. Prints [PASS], [WARN], or [FAIL] per item.
# Exits 1 on any FAIL.
set -euo pipefail

FAILS=0
WARNS=0
NS="${NAMESPACE:-default}"

pass() { echo "[PASS] $1"; }
warn() { echo "[WARN] $1"; WARNS=$((WARNS + 1)); }
fail() { echo "[FAIL] $1"; FAILS=$((FAILS + 1)); }

echo "=== Host Tuning ==="

# ---------- CPU governor ----------
CPUFREQ_DIR="/sys/devices/system/cpu/cpu0/cpufreq"
if [[ -d "$CPUFREQ_DIR" ]]; then
  BAD_GOVS=$(grep -cL performance /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor 2>/dev/null || true)
  if [[ -z "$BAD_GOVS" ]]; then
    pass "CPU governor = performance on all cores"
  else
    fail "CPU governor: some cores not set to performance"
  fi
else
  warn "CPU governor: cpufreq not available (VM without frequency scaling)"
fi

# ---------- Swap ----------
SWAP_TOTAL=$(awk '/SwapTotal/{print $2}' /proc/meminfo 2>/dev/null || echo "unknown")
if [[ "$SWAP_TOTAL" == "0" ]]; then
  pass "Swap disabled"
elif [[ "$SWAP_TOTAL" == "unknown" ]]; then
  warn "Swap: cannot read /proc/meminfo"
else
  fail "Swap still enabled (${SWAP_TOTAL} kB)"
fi

# ---------- vm.swappiness ----------
SWAPPINESS=$(sysctl -n vm.swappiness 2>/dev/null || echo "unknown")
if [[ "$SWAPPINESS" == "0" ]]; then
  pass "vm.swappiness = 0"
elif [[ "$SWAPPINESS" == "unknown" ]]; then
  warn "vm.swappiness: cannot read value"
else
  fail "vm.swappiness = ${SWAPPINESS} (expected 0)"
fi

# ---------- THP ----------
THP_ENABLED="/sys/kernel/mm/transparent_hugepage/enabled"
if [[ -f "$THP_ENABLED" ]]; then
  THP_VAL=$(cat "$THP_ENABLED" 2>/dev/null)
  if echo "$THP_VAL" | grep -q '\[madvise\]'; then
    pass "THP = madvise"
  else
    fail "THP = ${THP_VAL} (expected [madvise])"
  fi
else
  warn "THP: sysfs knob not present"
fi

# ---------- NTP ----------
NTP_OK=false
if command -v chronyc &>/dev/null; then
  if chronyc tracking 2>/dev/null | grep -q 'Leap status.*Normal'; then
    NTP_OK=true
  fi
elif command -v timedatectl &>/dev/null; then
  if timedatectl show --property=NTPSynchronized --value 2>/dev/null | grep -qi yes; then
    NTP_OK=true
  fi
fi
if $NTP_OK; then
  pass "NTP synchronized"
else
  warn "NTP: could not confirm clock synchronization"
fi

# ---------- k8s checks ----------
if ! command -v kubectl &>/dev/null; then
  warn "kubectl not found -- skipping k8s checks"
  echo ""
  echo "--- Summary ---"
  echo "FAIL: ${FAILS}  WARN: ${WARNS}"
  [[ "$FAILS" -gt 0 ]] && exit 1 || exit 0
fi

echo ""
echo "=== Experiment Infrastructure ==="

# All pods running
BAD_PODS=$(kubectl -n "$NS" get pods --no-headers 2>/dev/null \
  | awk '$3 !~ /^(Running|Completed)$/ {print $1}')
if [[ -z "$BAD_PODS" ]]; then
  pass "All pods Running/Completed"
else
  fail "Pods not ready: ${BAD_PODS//$'\n'/, }"
fi

# COLLECTOR_SERVICE_ADDR on all service deployments (trace completeness)
MISSING_COLLECTOR=""
SERVICES="adservice cartservice checkoutservice currencyservice emailservice frontend paymentservice productcatalogservice recommendationservice shippingservice"
for SVC in $SERVICES; do
  HAS_IT=$(kubectl -n "$NS" get deployment "$SVC" -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="COLLECTOR_SERVICE_ADDR")].value}' 2>/dev/null || true)
  if [[ -z "$HAS_IT" ]]; then
    MISSING_COLLECTOR="${MISSING_COLLECTOR} ${SVC}"
  fi
done
if [[ -z "$MISSING_COLLECTOR" ]]; then
  pass "COLLECTOR_SERVICE_ADDR set on all services"
else
  fail "COLLECTOR_SERVICE_ADDR missing on:${MISSING_COLLECTOR}"
fi

# Guaranteed QoS (requests == limits) — scheduling jitter confound
BAD_QOS=""
for SVC in $SERVICES; do
  QOS=$(kubectl -n "$NS" get pod -l app="$SVC" -o jsonpath='{.items[0].status.qosClass}' 2>/dev/null || echo "unknown")
  if [[ "$QOS" != "Guaranteed" ]]; then
    BAD_QOS="${BAD_QOS} ${SVC}(${QOS})"
  fi
done
if [[ -z "$BAD_QOS" ]]; then
  pass "All service pods have Guaranteed QoS"
else
  fail "Non-Guaranteed QoS:${BAD_QOS}"
fi

# No HPA (replica count must be fixed across phases)
HPA_COUNT=$(kubectl -n "$NS" get hpa --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [[ "$HPA_COUNT" == "0" ]]; then
  pass "No HPA active (fixed replica counts)"
else
  fail "${HPA_COUNT} HPA(s) active — replica count will drift between phases"
fi

# AlwaysSample tracing (ratio sampling invalidates latency distributions)
# Check the otel-collector config for a probabilistic sampler
COLLECTOR_CFG=$(kubectl -n "$NS" get configmap otel-collector-config -o jsonpath='{.data}' 2>/dev/null || true)
if [[ -n "$COLLECTOR_CFG" ]]; then
  if echo "$COLLECTOR_CFG" | grep -qi "probabilistic"; then
    warn "OTel collector config contains probabilistic sampler — latency distributions may be biased"
  else
    pass "No probabilistic sampler in OTel collector config"
  fi
else
  warn "Could not read otel-collector configmap"
fi

# ---------- Summary ----------
echo ""
echo "--- Summary ---"
echo "FAIL: ${FAILS}  WARN: ${WARNS}"
if [[ "$FAILS" -gt 0 ]]; then
  echo "Fix the above before running experiments."
  echo "Reminder: run ../reset-online-boutique-state.sh between phases."
  exit 1
else
  echo "Ready for experiment."
  echo "Reminder: run ../reset-online-boutique-state.sh between phases."
  exit 0
fi
