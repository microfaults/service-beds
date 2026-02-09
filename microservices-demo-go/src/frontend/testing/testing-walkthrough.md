# Testing Walkthrough - Frontend Service

This guide explains how to verify the new Go/REST Frontend service using the included Mock Server.

## Prerequisites
- **Go 1.25+** (if running locally)
- **Docker** (if running via containers)
- Terminal Access

## Quick Start (Run Locally)

### 1. Build & Run the Mock Server
The mock server simulates the backend services (ProductCatalog, Cart, Currency, etc).

```bash
cd microservices-demo-go/src/frontend

# Build the mock server from its new location
go build -o mock_server testing/mock_server/main.go

# Run it on port 8081
export PORT=8081
./mock_server
```
*Leave this terminal running.*

### 2. Build & Run the Frontend
Open a **new terminal window** and navigate to the same directory.

```bash
cd microservices-demo-go/src/frontend

# Build the frontend service
go build -o frontend .

# Configure frontend to talk to the mock server (localhost:8081)
export PORT=8080
export PRODUCT_CATALOG_SERVICE_ADDR="localhost:8081"
export CURRENCY_SERVICE_ADDR="localhost:8081"
export CART_SERVICE_ADDR="localhost:8081"
export RECOMMENDATION_SERVICE_ADDR="localhost:8081"
export CHECKOUT_SERVICE_ADDR="localhost:8081"
export SHIPPING_SERVICE_ADDR="localhost:8081"
export AD_SERVICE_ADDR="localhost:8081"
export SHOPPING_ASSISTANT_SERVICE_ADDR="localhost:8081"

# Run the frontend
./frontend
```

### 3. Verify in Browser
Open [http://localhost:8080](http://localhost:8080) in your web browser.

**What to check:**
- [ ] **Home Page**: You should see products "Mug" and "Tank Top".
- [ ] **Product Details**: Click on a product. You should see its description and price.
- [ ] **Add to Cart**: Click "Add to Cart". You should be redirected to the cart page with the item listed.
- [ ] **Checkout**: Click "Place Order". You should see a successful order confirmation.
