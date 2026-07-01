#!/bin/sh
# ---------------------------------------------------------------------------
# Online Boutique (microservices-demo-go) — endpoint + e2e smoke test suite
# Runs INSIDE an in-cluster pod that has `curl` (ClusterIP DNS reachable).
# Contracts derived from src/frontend/clients/clients.go + each service's
# atropos.RegisterRoutes() route inventory (develop @ bfe226a).
# Exit code = number of failed checks.
# ---------------------------------------------------------------------------
PASS=0; FAIL=0; FAILED_NAMES=""
PID="OLJCESPC7Z"       # Sunglasses (known-good product id)
PID2="L9ECAV7KIM"
CART="smoke-$(date +%s)"

# check NAME EXPECTED_CODE SUBSTR METHOD URL [CONTENT_TYPE] [BODY]
check() {
  name="$1"; exp="$2"; sub="$3"; method="$4"; url="$5"; ct="$6"; body="$7"
  if [ -n "$body" ]; then
    out=$(curl -s -m 15 -X "$method" -H "Content-Type: ${ct:-application/json}" --data "$body" -w '\n%{http_code}' "$url" 2>/dev/null)
  else
    out=$(curl -s -m 15 -X "$method" -w '\n%{http_code}' "$url" 2>/dev/null)
  fi
  code=$(printf '%s' "$out" | awk 'END{print}')
  resp=$(printf '%s' "$out" | sed '$d')
  ok=1
  [ "$code" = "$exp" ] || ok=0
  if [ -n "$sub" ]; then printf '%s' "$resp" | grep -q "$sub" 2>/dev/null || ok=0; fi
  if [ "$ok" = "1" ]; then
    PASS=$((PASS+1)); printf 'PASS  %-46s HTTP %s\n' "$name" "$code"
  else
    FAIL=$((FAIL+1)); FAILED_NAMES="$FAILED_NAMES\n   - $name"
    printf 'FAIL  %-46s HTTP %s (want %s%s) body: %s\n' "$name" "$code" "$exp" "$( [ -n "$sub" ] && echo " + /$sub/")" "$(printf '%s' "$resp" | head -c 100)"
  fi
}

echo "================ PRODUCTCATALOGSERVICE (:3550) ================"
check "productcatalog GET /products"            200 '"products"'  GET  "http://productcatalogservice:3550/products"
check "productcatalog GET /products/{id}"       200 'Sunglasses'  GET  "http://productcatalogservice:3550/products/$PID"
check "productcatalog GET /products/{bad} 404"  404 ''            GET  "http://productcatalogservice:3550/products/NOSUCHID"
check "productcatalog GET /products/search?q"   200 '"results"'   GET  "http://productcatalogservice:3550/products/search?q=watch"
check "productcatalog POST /products/batch"     200 '"products"'  POST "http://productcatalogservice:3550/products/batch" application/json "{\"ids\":[\"$PID\",\"$PID2\"]}"

echo "================ CURRENCYSERVICE (:7000) ================"
check "currency GET /currencies"                200 'EUR'            GET  "http://currencyservice:7000/currencies"
check "currency POST /convert"                  200 'currency_code'  POST "http://currencyservice:7000/convert" application/json "{\"from\":{\"currency_code\":\"USD\",\"units\":100,\"nanos\":0},\"to_code\":\"EUR\"}"

echo "================ CARTSERVICE (:7070) ================"
check "cart DELETE /cart/{u} (pre-clean)"       200 ''            DELETE "http://cartservice:7070/cart/$CART"
check "cart POST /cart/{u}/items"               200 ''            POST "http://cartservice:7070/cart/$CART/items" application/json "{\"product_id\":\"$PID\",\"quantity\":3}"
check "cart GET /cart/{u} (has item)"           200 "$PID"        GET  "http://cartservice:7070/cart/$CART"
check "cart DELETE /cart/{u} (empty)"           200 ''            DELETE "http://cartservice:7070/cart/$CART"

echo "================ SHIPPINGSERVICE (:50051) ================"
check "shipping POST /shipping/quote"           200 'cost_usd'    POST "http://shippingservice:50051/shipping/quote" application/json "{\"items\":[{\"product_id\":\"$PID\",\"quantity\":3}],\"currency\":\"USD\"}"
check "shipping POST /shipping/ship"            200 'tracking'    POST "http://shippingservice:50051/shipping/ship" application/json "{\"address\":{\"street_address\":\"1600 Amphitheatre Pkwy\",\"city\":\"Mountain View\",\"state\":\"CA\",\"country\":\"USA\",\"zip_code\":94043},\"items\":[{\"product_id\":\"$PID\",\"quantity\":3}]}"

echo "================ RECOMMENDATIONSERVICE (:8080) ================"
check "recommendation GET /recommendations"     200 'product_ids' GET  "http://recommendationservice:8080/recommendations?user_id=smoke&product_ids=$PID"

echo "================ ADSERVICE (:9555) ================"
check "ad GET /ads?context_keys=clothing"       200 '"ads"'       GET  "http://adservice:9555/ads?context_keys=clothing"
check "ad GET /ads (no keys -> random)"         200 '"ads"'       GET  "http://adservice:9555/ads"
check "ad POST /ads (405 - GET-only contract)"  405 ''            POST "http://adservice:9555/ads" application/json "{\"context_keys\":[\"x\"]}"

echo "================ PAYMENTSERVICE (:50051) ================"
check "payment POST /charge"                    200 'transaction_id' POST "http://paymentservice:50051/charge" application/json "{\"amount\":{\"currency_code\":\"USD\",\"units\":50,\"nanos\":0},\"credit_card\":{\"credit_card_number\":\"4432801561520454\",\"credit_card_cvv\":672,\"credit_card_expiration_year\":2030,\"credit_card_expiration_month\":1}}"

echo "================ EMAILSERVICE (:5000) ================"
check "email POST /send-order-confirmation"     200 ''            POST "http://emailservice:5000/send-order-confirmation" application/json "{\"email\":\"smoke@example.com\",\"order\":{\"order_id\":\"smoke-1\",\"shipping_tracking_id\":\"track-1\",\"shipping_cost\":{\"currency_code\":\"USD\",\"units\":5,\"nanos\":0},\"shipping_address\":{\"street_address\":\"1600 Amphitheatre Pkwy\",\"city\":\"Mountain View\",\"state\":\"CA\",\"country\":\"USA\",\"zip_code\":94043},\"items\":[]}}"

echo "================ CHECKOUTSERVICE (:5050) — full order e2e ================"
CO="smoke-co-$(date +%s)"
check "  (setup) cart add for checkout user"    200 ''            POST "http://cartservice:7070/cart/$CO/items" application/json "{\"product_id\":\"$PID\",\"quantity\":1}"
check "checkout POST /placeorder"               200 'order_id'    POST "http://checkoutservice:5050/placeorder" application/json "{\"user_id\":\"$CO\",\"user_currency\":\"USD\",\"email\":\"smoke@example.com\",\"address\":{\"street_address\":\"1600 Amphitheatre Pkwy\",\"city\":\"Mountain View\",\"state\":\"CA\",\"country\":\"USA\",\"zip_code\":94043},\"credit_card\":{\"credit_card_number\":\"4432801561520454\",\"credit_card_cvv\":672,\"credit_card_expiration_year\":2030,\"credit_card_expiration_month\":1}}"

echo "================ FRONTEND (:80) — user-facing e2e flows (session cookie) ================"
J=/tmp/cookies.txt; rm -f $J
check "frontend GET /_healthz"                  200 ''            GET  "http://frontend:80/_healthz"
# home + product page (session cookie set on first GET /)
curl -s -m 15 -c $J -o /dev/null "http://frontend:80/" 2>/dev/null
check "frontend GET / (home listing)"           200 'Hot Products\|hipstershop\|/product/' GET "http://frontend:80/"
check "frontend GET /product/{id}"              200 'Add To Cart\|Sunglasses'              GET "http://frontend:80/product/$PID"
check "frontend POST /setCurrency (302 redir)"  302 ''            POST "http://frontend:80/setCurrency" application/x-www-form-urlencoded "currency_code=EUR"
# cart add + view (use cookie jar so session cart persists)
curl -s -m 15 -b $J -c $J -o /dev/null -X POST -H "Content-Type: application/x-www-form-urlencoded" --data "product_id=$PID&quantity=1" "http://frontend:80/cart" 2>/dev/null
FCART=$(curl -s -m 15 -b $J -c $J -w '\n%{http_code}' "http://frontend:80/cart" 2>/dev/null)
fcode=$(printf '%s' "$FCART" | awk 'END{print}'); fbody=$(printf '%s' "$FCART" | sed '$d')
if [ "$fcode" = "200" ] && printf '%s' "$fbody" | grep -q "Shopping Cart\|Shipping\|Total Cost\|$PID"; then
  PASS=$((PASS+1)); printf 'PASS  %-46s HTTP %s\n' "frontend GET /cart (item + shipping est)" "$fcode"
else
  FAIL=$((FAIL+1)); FAILED_NAMES="$FAILED_NAMES\n   - frontend GET /cart"
  printf 'FAIL  %-46s HTTP %s body: %s\n' "frontend GET /cart" "$fcode" "$(printf '%s' "$fbody" | head -c 100)"
fi
# checkout via the browser form (creates a real order through the whole chain)
CKOUT=$(curl -s -m 20 -b $J -c $J -w '\n%{http_code}' -X POST -H "Content-Type: application/x-www-form-urlencoded" \
  --data "email=smoke@example.com&street_address=1600+Amphitheatre+Pkwy&zip_code=94043&city=Mountain+View&state=CA&country=USA&credit_card_number=4432801561520454&credit_card_expiration_month=1&credit_card_expiration_year=2030&credit_card_cvv=672" \
  "http://frontend:80/cart/checkout" 2>/dev/null)
ckcode=$(printf '%s' "$CKOUT" | awk 'END{print}'); ckbody=$(printf '%s' "$CKOUT" | sed '$d')
if [ "$ckcode" = "200" ] && printf '%s' "$ckbody" | grep -q "Order Confirmation\|order-id\|Tracking\|confirmed\|Thank"; then
  PASS=$((PASS+1)); printf 'PASS  %-46s HTTP %s\n' "frontend POST /cart/checkout (order placed)" "$ckcode"
else
  FAIL=$((FAIL+1)); FAILED_NAMES="$FAILED_NAMES\n   - frontend POST /cart/checkout"
  printf 'FAIL  %-46s HTTP %s body: %s\n' "frontend POST /cart/checkout" "$ckcode" "$(printf '%s' "$ckbody" | head -c 120)"
fi
check "frontend GET /product-meta/{ids}"        200 ''            GET "http://frontend:80/product-meta/$PID"
check "frontend GET /assistant"                 200 ''            GET "http://frontend:80/assistant"

echo ""
echo "================================================================"
printf 'RESULT: %s passed, %s failed\n' "$PASS" "$FAIL"
[ "$FAIL" -gt 0 ] && printf 'Failed checks:%b\n' "$FAILED_NAMES"
exit $FAIL
