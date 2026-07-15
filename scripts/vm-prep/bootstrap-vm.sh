#!/usr/bin/env bash
# bootstrap-vm.sh — provision a FRESH Ubuntu/Debian VM to run the faults-lab
# testbed the way vm1 (2262-cse115b-01) actually runs it:
#
#   skaffold --(docker build)--> local registry:2 on :5000
#                                     |
#   k3s (containerd runtime) <--- registries.yaml mirror (http://localhost:5000)
#
# This replaces the retired vm-setup.sh (chore/vm-readiness), which
# bootstrapped a topology the live testbed never used (k3s --docker,
# traefik disabled, no registry).
#
# DO NOT run on the live vm1 testbed: it may restart k3s to pick up
# registries.yaml. For host tuning on an already-provisioned VM, use
# ./prepare-vm.sh instead.
#
# Usage: sudo ./bootstrap-vm.sh
#
# After it finishes:
#   1. clone manteion-go and service-beds (vm1 keeps them under /home/faults-lab/)
#   2. sudo ./prepare-vm.sh                       # host tuning for fidelity
#   3. deploy manteion FIRST (its own skaffold.yaml) — services fail hard at
#      startup when MANTEION_URL is set but unreachable (atropos-go >= v0.1.0)
#   4. cd service-beds/microservices-demo-go && skaffold run
#   5. ./verify-readiness.sh before every experiment
set -euo pipefail

info()  { printf '\033[1;34m[info]\033[0m  %s\n' "$1"; }
error() { printf '\033[1;31m[error]\033[0m %s\n' "$1"; exit 1; }

[[ $EUID -eq 0 ]] || error "Run as root (or with sudo)."

REAL_USER="${SUDO_USER:-$USER}"
REAL_HOME=$(eval echo "~$REAL_USER")

info "Updating package index"
apt-get update -qq

# ---------- [1/6] Docker (build engine for skaffold) ----------
if command -v docker &>/dev/null; then
  info "Docker already installed: $(docker --version)"
else
  info "Installing Docker"
  apt-get install -y -qq ca-certificates curl gnupg
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
    | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
     https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update -qq
  apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin
  info "Docker installed: $(docker --version)"
fi

# ---------- [2/6] Local image registry on :5000 ----------
# skaffold pushes here; k3s pulls from here. restart=always matches vm1.
if docker inspect registry &>/dev/null; then
  docker start registry >/dev/null 2>&1 || true
  info "Local registry container already exists (ensured running)"
else
  info "Starting local registry:2 on :5000"
  docker run -d --name registry --restart=always -p 5000:5000 registry:2
fi

# ---------- [3/6] k3s registries.yaml (BEFORE k3s starts) ----------
# containerd needs an explicit mirror to pull from the plain-HTTP local
# registry. Written before install so a fresh k3s reads it on first boot.
REG_FILE="/etc/rancher/k3s/registries.yaml"
REG_CONTENT='mirrors:
  "localhost:5000":
    endpoint:
      - "http://localhost:5000"'
mkdir -p /etc/rancher/k3s
NEED_K3S_RESTART=0
if [[ -f "$REG_FILE" ]] && diff -q <(printf '%s\n' "$REG_CONTENT") "$REG_FILE" >/dev/null 2>&1; then
  info "registries.yaml already correct"
else
  printf '%s\n' "$REG_CONTENT" > "$REG_FILE"
  info "Wrote $REG_FILE"
  NEED_K3S_RESTART=1
fi

# ---------- [4/6] k3s (containerd runtime — NOT --docker) ----------
if command -v k3s &>/dev/null; then
  info "k3s already installed: $(k3s --version | head -1)"
  if [[ "$NEED_K3S_RESTART" -eq 1 ]]; then
    info "Restarting k3s to pick up registries.yaml"
    systemctl restart k3s
  fi
else
  info "Installing k3s (default containerd runtime, traefik enabled — matches vm1)"
  curl -sfL https://get.k3s.io | sh -
fi
info "Waiting for k3s node to be ready"
until k3s kubectl get nodes 2>/dev/null | grep -q ' Ready'; do sleep 2; done
info "k3s is ready"

# ---------- [5/6] kubectl + kubeconfig for $REAL_USER ----------
if ! command -v kubectl &>/dev/null; then
  ln -sf /usr/local/bin/k3s /usr/local/bin/kubectl
  info "Symlinked kubectl -> k3s"
fi
if [[ -f /etc/rancher/k3s/k3s.yaml ]]; then
  mkdir -p "$REAL_HOME/.kube"
  cp /etc/rancher/k3s/k3s.yaml "$REAL_HOME/.kube/config"
  chown -R "$REAL_USER" "$REAL_HOME/.kube"
  info "Copied kubeconfig to $REAL_HOME/.kube/config"
fi

# ---------- [6/6] skaffold, wired to the local registry ----------
if command -v skaffold &>/dev/null; then
  info "Skaffold already installed"
else
  info "Installing Skaffold"
  ARCH=$(uname -m)
  case "$ARCH" in
    x86_64)  SKAFFOLD_ARCH="amd64" ;;
    aarch64) SKAFFOLD_ARCH="arm64" ;;
    *)       error "Unsupported architecture: $ARCH" ;;
  esac
  curl -Lo /usr/local/bin/skaffold \
    "https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-${SKAFFOLD_ARCH}"
  chmod +x /usr/local/bin/skaffold
fi
# default-repo makes every skaffold build push to the local registry — this
# is the glue between the docker build side and the containerd pull side.
sudo -u "$REAL_USER" skaffold config set default-repo localhost:5000 -k default
info "skaffold default-repo=localhost:5000 for kube-context 'default'"

# ---------- verify ----------
info "Verification:"
echo "  docker:    $(docker --version)"
echo "  registry:  $(curl -s localhost:5000/v2/_catalog || echo UNREACHABLE)"
echo "  k3s:       $(k3s --version | head -1)"
echo "  skaffold:  $(skaffold version 2>/dev/null || echo installed)"
echo ""
info "Next: clone manteion-go + service-beds, run 'sudo ./prepare-vm.sh',"
info "deploy manteion FIRST, then 'cd microservices-demo-go && skaffold run',"
info "and gate experiments with ./verify-readiness.sh."
