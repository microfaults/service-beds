# VM Experiment Preparation

Scripts to minimize confounds when running faults-lab experiments on a k3s VM.

## Quick start

```bash
# One-time setup (run as root on the experiment VM):
sudo ./prepare-vm.sh

# Before each experiment run:
./verify-readiness.sh
```

## prepare-vm.sh

Must be run as root. Applies kernel and scheduler tuning that persists across
reboots where possible:

1. **CPU governor -> performance** -- pins all cores to max frequency so
   latency measurements aren't skewed by DVFS. Gracefully skips if cpufreq
   isn't exposed (common in VMs).
2. **Swap off** -- calls `swapoff -a` and comments out swap entries in
   `/etc/fstab` so the OOM-killer fires instead of silently paging.
3. **vm.swappiness=0** -- set via sysctl and persisted in `/etc/sysctl.conf`.
4. **THP -> madvise** -- disables transparent huge pages for all allocations
   except those that explicitly opt in. Skips if the sysfs knob isn't present.
5. **NTP sync check** -- verifies the clock is synchronized (tries chronyc
   first, falls back to timedatectl). Does not fix drift, only warns.

## verify-readiness.sh

Non-destructive checks to run before each experiment. Reports [PASS], [WARN],
or [FAIL] for each item:

- CPU governor = performance
- Swap disabled
- vm.swappiness = 0
- THP = madvise
- NTP synchronized
- All k8s pods Running or Completed
- COLLECTOR_SERVICE_ADDR set on all service deployments
- No deployments with imagePullPolicy=Always

Exits 0 if all checks pass or warn. Exits 1 if any check fails.

## Confounds NOT addressed by these scripts

The following require manual attention:

- **Node topology / loadgen separation** -- the load generator should not
  compete for CPU with the services under test. Pin it to a separate node or
  use taints/tolerations.
- **Background workloads** -- ensure no unrelated pods or cron jobs are
  running during the experiment window.
- **Network** -- VM-to-VM network jitter, bandwidth limits, and MTU settings
  are outside the scope of these scripts.
- **Sampling config** -- trace/metric sampling rates must be set consistently
  across experiments. Check the otel-collector config and per-service SDK
  settings.
