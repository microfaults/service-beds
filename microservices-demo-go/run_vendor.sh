#!/bin/bash
export PATH=${GO_BIN_DIR:-/usr/local/go/bin}:$PATH
set -e

export GOPRIVATE="git.ucsc.edu/microfaults/*"

SERVICES=(
  "src/emailservice"
  "src/productcatalogservice"
  "src/recommendationservice"
  "src/shoppingassistantservice"
  "src/shippingservice"
  "src/checkoutservice"
  "src/paymentservice"
  "src/currencyservice"
  "src/cartservice/src"
  "src/frontend"
  "src/adservice"
)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

for svc in "${SERVICES[@]}"; do
  dir="$SCRIPT_DIR/$svc"
  if [ -f "$dir/go.mod" ]; then
    echo "==> Vendoring $svc"
    (cd "$dir" && go mod vendor)
  else
    echo "==> Skipping $svc (no go.mod found)"
  fi
done

echo "==> All services vendored successfully"
