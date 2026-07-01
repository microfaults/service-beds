# Endpoint + E2E smoke tests

`endpoint_e2e_test.sh` exercises every backend service endpoint (contracts taken
from each service's `atropos.RegisterRoutes()` inventory and `frontend/clients`)
plus the user-facing frontend flows (home, product, currency, cart, checkout,
assistant) with a session cookie. Assertions check HTTP status + a key response
field. Exit code = number of failed checks.

## Run (against the in-cluster stack)

```sh
# from a machine with kubectl access to the boutique namespace (default):
kubectl run tester --image=curlimages/curl:8.10.1 -n default --restart=Never --command -- sleep 7200
kubectl exec -i tester -n default -- sh -s < microservices-demo-go/tests/endpoint_e2e_test.sh
kubectl delete pod tester -n default
```

Requires the services reachable by their ClusterIP DNS names/ports
(productcatalogservice:3550, currencyservice:7000, cartservice:7070,
shippingservice:50051, recommendationservice:8080, adservice:9555,
paymentservice:50051, emailservice:5000, checkoutservice:5050, frontend:80).

## Coverage notes / regression guards
- `ad POST /ads` must return 405 (service is GET-only: `GET /ads?context_keys=`).
- `email POST /send-order-confirmation` uses int `zip_code` (canonical demo.proto
  type); a 400 here means a service reintroduced the string/int schema drift.
- `checkout POST /placeorder` runs the full price->charge->ship->email->empty chain.
- Frontend `POST /setCurrency` returns 302 (redirect), not 200.
- Frontend checkout requires an **undashed** card number (validator: `credit_card`).
