#!/usr/bin/env bash
# Tunes a Linux VM for reproducible experiment runs.
# Must be run as root. Idempotent -- safe to re-run.
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "error: must run as root" >&2
  exit 1
fi

# ---------- [1/5] CPU governor -> performance ----------
echo "[1/5] CPU governor"
CPUFREQ_DIR="/sys/devices/system/cpu/cpu0/cpufreq"
if [[ -d "$CPUFREQ_DIR" ]]; then
  for gov in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do
    echo performance > "$gov"
  done
  echo "  set all cores to performance"
else
  echo "  cpufreq not available (VM without frequency scaling) -- skipped"
fi

# ---------- [2/5] Disable swap ----------
echo "[2/5] Swap"
swapoff -a
echo "  swap disabled"
# Comment out swap lines in fstab (idempotent: skip already-commented lines)
if grep -qE '^[^#].*\sswap\s' /etc/fstab 2>/dev/null; then
  sed -i.bak '/^[^#].*\sswap\s/s/^/#/' /etc/fstab
  echo "  commented swap entries in /etc/fstab"
else
  echo "  /etc/fstab already has no active swap entries"
fi

# ---------- [3/5] vm.swappiness=0 ----------
echo "[3/5] vm.swappiness"
sysctl -w vm.swappiness=0 >/dev/null
# Persist: update existing line or append
if grep -q '^vm.swappiness' /etc/sysctl.conf 2>/dev/null; then
  sed -i.bak 's/^vm.swappiness=.*/vm.swappiness=0/' /etc/sysctl.conf
else
  echo 'vm.swappiness=0' >> /etc/sysctl.conf
fi
echo "  vm.swappiness=0 (live + persisted)"

# ---------- [4/5] THP -> madvise ----------
echo "[4/5] Transparent Huge Pages"
THP_ENABLED="/sys/kernel/mm/transparent_hugepage/enabled"
if [[ -f "$THP_ENABLED" ]]; then
  echo madvise > "$THP_ENABLED"
  echo "  THP set to madvise"
else
  echo "  THP sysfs knob not present -- skipped"
fi

# ---------- [5/5] NTP sync check ----------
echo "[5/5] NTP synchronization"
if command -v chronyc &>/dev/null; then
  if chronyc tracking 2>/dev/null | grep -q 'Leap status.*Normal'; then
    echo "  clock synchronized (chrony)"
  else
    echo "  WARNING: chrony reports clock may not be synchronized"
  fi
elif command -v timedatectl &>/dev/null; then
  if timedatectl show --property=NTPSynchronized --value 2>/dev/null | grep -qi yes; then
    echo "  clock synchronized (timedatectl)"
  else
    echo "  WARNING: timedatectl reports clock is not synchronized"
  fi
else
  echo "  WARNING: neither chronyc nor timedatectl found -- cannot verify NTP"
fi

echo ""
echo "VM preparation complete."
