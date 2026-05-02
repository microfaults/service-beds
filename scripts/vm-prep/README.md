# VM Experiment Preparation

Scripts to minimize confounds when running faults-lab cache-box isolation
experiments on a k3s VM. Grounded in the confound list from `VISION.md`
and the interference hierarchy (L0–L4) from `atropos-go/VISION.md`.

## Quick start

```bash
# One-time host tuning (run as root on the experiment VM):
sudo ./prepare-vm.sh

# Before each experiment run:
./verify-readiness.sh

# Between experiment phases (baseline → 1a → 1b → 2a → …):
../reset-online-boutique-state.sh
```

## prepare-vm.sh

Must be run as root. Applies kernel tuning that persists across reboots:

1. **CPU governor → performance** — pins all cores to max frequency so
   latency measurements aren't skewed by DVFS ramp-up. Skips gracefully
   if cpufreq isn't exposed (common in VMs).
2. **Swap off** — `swapoff -a` + comments out fstab entries. OOM-kill is
   preferable to silent paging during experiments.
3. **vm.swappiness=0** — sysctl live + persisted.
4. **THP → madvise** — avoids compaction stalls from transparent huge
   pages. Skips if the sysfs knob isn't present.
5. **NTP sync check** — verifies the clock is synchronized (tries chronyc,
   falls back to timedatectl). Clock skew corrupts trace span ordering.

## verify-readiness.sh

Non-destructive pre-experiment checks. Reports `[PASS]`, `[WARN]`, or
`[FAIL]` for each item. Exits 1 on any FAIL.

**Host checks:** CPU governor, swap, swappiness, THP, NTP.

**Experiment infrastructure checks:**
- All pods Running/Completed
- COLLECTOR_SERVICE_ADDR set on all service deployments (traces silently
  drop without it — the #1 confound we hit)
- Guaranteed QoS on all service pods (requests == limits, so the scheduler
  doesn't throttle or over-commit)
- No HPA active (replica count must be fixed across phases)
- AlwaysSample tracing (ratio sampling invalidates latency distributions)

## Known confounds (from VISION.md)

These are the documented confounds for cache-box isolation experiments.
The scripts address what they can; the rest requires experimental design.

| Confound | Status | Mitigation |
|---|---|---|
| Closed-loop generator inflates throughput when service is cached | **Fixed** | Zeus k6 uses `constant-arrival-rate` (open-loop). Invalid if `dropped_iterations > 0`. |
| State drift across phases (cart data, Kafka offsets) | **Fixed** | Run `reset-online-boutique-state.sh` between phases. Script flushes Redis, rolls cartservice, rolls Kafka. |
| p50-only stub in replay-with-delay erases second moment | **Known** | Biases Δ(2−4) toward smaller magnitudes. Fix: empirical-CDF sampling from baseline traces. Tracked in atropos-go. |
| L3 co-locator interference: frozen service frees CPU/mem, neighbors speed up | **Known** | Over-attributes savings to frozen target. Mitigate with randomized pod-to-node placement across replicate runs. |
| Single-run point estimates | **Known** | Replicate each phase ≥3 times. Latin-square phase ordering if sequential order effects are suspected. |
| Cache hit rate fidelity | **Known** | Verify via `GET /admin/cachebox` stats on each service after a cache-seed phase. Low hit rate means the keying strategy doesn't match the load profile. |

## What scripts cannot check

- **Phase reset was run** — scripts can't know which phase you're about to
  run. Run `reset-online-boutique-state.sh` yourself between phases.
- **Zeus workflow is open-loop** — verify `constant-arrival-rate` executor
  in the k6 script / zeus flow JSON.
- **Pod placement** — for L3 mitigation, randomize pod-to-node mapping
  across replicate runs (pod anti-affinity or random scheduler). Record
  the placement in `ExperimentRun.NodePlacement`.
