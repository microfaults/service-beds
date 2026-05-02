#!/usr/bin/env bash
# Pre-experiment readiness check. Non-destructive.
# Prints [PASS], [WARN], or [FAIL] for each item.
# Exits 1 if any FAIL.
set -euo pipefail

FAILS=0
WARNS=0
NS="${NAMESPACE:-default}"

pass() { echo "[PASS] $1"; }
warn() { echo "[WARN] $1"; WARNS=$((WARNS + 1)); }
fail() { echo "[FAIL] $1"; FAILS=$((FAILS + 1)); }

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

# ---------- k8s: pod status ----------
if command -v kubectl &>/dev/null; then
  BAD_PODS=$(kubectl -n "$NS" get pods --no-headers 2>/dev/null \
    | awk '$3 !~ /^(Running|Completed)$/ {print $1}')
  if [[ -z "$BAD_PODS" ]]; then
    pass "All pods Running/Completed"
  else
    fail "Pods not Running/Completed: ${BAD_PODS//$'\n'/, }"
  fi

  # ---------- COLLECTOR_SERVICE_ADDR ----------
  MISSING_COLLECTOR=""
  for DEPLOY in $(kubectl -n "$NS" get deployments -o name 2>/dev/null); do
    DEPLOY_NAME="${DEPLOY#deployment.apps/}"
    # Check if any container in this deployment references COLLECTOR_SERVICE_ADDR
    HAS_COLLECTOR=$(kubectl -n "$NS" get "$DEPLOY" -o jsonpath='{.spec.template.spec.containers[*].env[*].name}' 2>/dev/null || true)
    if echo "$HAS_COLLECTOR" | grep -q 'COLLECTOR_SERVICE_ADDR'; then
      continue
    fi
    # Also check envFrom / configMapRef -- but for simple detection, check the
    # rendered pod spec for the env var name anywhere in the deployment JSON.
    HAS_COLLECTOR_FULL=$(kubectl -n "$NS" get "$DEPLOY" -o json 2>/dev/null | grep -c 'COLLECTOR_SERVICE_ADDR' || true)
    if [[ "$HAS_COLLECTOR_FULL" -gt 0 ]]; then
      continue
    fi
    MISSING_COLLECTOR="${MISSING_COLLECTOR} ${DEPLOY_NAME}"
  done
  if [[ -z "$MISSING_COLLECTOR" ]]; then
    pass "COLLECTOR_SERVICE_ADDR set on all deployments"
  else
    warn "COLLECTOR_SERVICE_ADDR missing on:${MISSING_COLLECTOR}"
  fi

  # ---------- imagePullPolicy ----------
  ALWAYS_PULL=$(kubectl -n "$NS" get deployments -o json 2>/dev/null \
    | python3 -c '
import sys, json
data = json.load(sys.stdin)
bad = []
for d in data.get("items", []):
    name = d["metadata"]["name"]
    for c in d["spec"]["template"]["spec"].get("containers", []):
        if c.get("imagePullPolicy") == "Always":
            bad.append(f"{name}/{c['name']}")
if bad:
    print(", ".join(bad))
' 2>/dev/null || true)
  if [[ -z "$ALWAYS_PULL" ]]; then
    pass "No deployments with imagePullPolicy=Always"
  else
    fail "imagePullPolicy=Always on: ${ALWAYS_PULL}"
  fi
else
  warn "kubectl not found -- skipping k8s checks"
fi

# ---------- Summary ----------
echo ""
echo "--- Summary ---"
echo "FAIL: ${FAILS}  WARN: ${WARNS}"
if [[ "$FAILS" -gt 0 ]]; then
  echo "Readiness check FAILED. Fix the above before running experiments."
  exit 1
else
  echo "Ready for experiment."
  exit 0
fi
