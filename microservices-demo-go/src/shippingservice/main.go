// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.ucsc.edu/microfaults/atropos-go"

	"github.com/sirupsen/logrus"
)

const (
	defaultPort = "8080"
)

var log *logrus.Logger

func init() {
	log = logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout
}

func main() {
	ctx := context.Background()
	shutdown, err := atropos.Init(ctx,
		atropos.WithServiceName("shippingservice"),
		atropos.WithServiceVersion("0.1.0"),
	)
	if err != nil {
		log.Fatalf("failed to init atropos: %v", err)
	}
	defer shutdown(ctx)

	eval := atropos.NewStaticEvaluator()

	instanceID := resolveInstanceID("shippingservice")
	// No explicit Store: the SDK's default bounded RecordBuffer counts overflow
	// instead of evicting (an LRU MemStore silently drops recordings under load).
	cbCfg := atropos.CacheBoxConfig{}
	cbPush := newCachePush("shippingservice", instanceID)
	if cbPush != nil {
		cbCfg.Push = cbPush.PushFunc()
	}

	cb := atropos.NewCacheBox(cbCfg)
	// Push-side fidelity counts must land in the same registry the drain report
	// snapshots; bind before traffic flows.
	if cbPush != nil {
		cbPush.BindFidelity(cb.Fidelity())
	}
	atropos.Configure(
		atropos.WithEvaluator(eval),
		atropos.WithCacheBoxCoordinator(cb),
	)

	atropos.RegisterRoutes(
		atropos.Route{Method: "POST", Path: "/shipping/quote", Description: "Get a shipping quote for an address and item list"},
		atropos.Route{Method: "POST", Path: "/shipping/ship", Description: "Ship an order; returns a tracking ID"},
	)

	applyTargets := atropos.ApplyTargets{Evaluator: eval, CacheBox: cb}
	if cbPush != nil {
		applyTargets.CacheDrain = atropos.NewCacheDrainTracker(cb, cbPush, nil)
	}
	mc, err := atropos.ConnectManteion(ctx, "shippingservice",
		atropos.WithInstanceID(instanceID),
		atropos.WithApplyTargets(applyTargets),
	)
	if err != nil {
		log.Warnf("manteion connection failed, running offline: %v", err)
	}
	if mc != nil {
		defer mc.Close(ctx)
	}
	if cbPush != nil {
		defer cbPush.Stop()
	}

	port := defaultPort
	if value, ok := os.LookupEnv("PORT"); ok {
		port = value
	}
	addr := fmt.Sprintf(":%s", port)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /shipping/quote", handleGetQuote)
	mux.HandleFunc("POST /shipping/ship", handleShipOrder)
	mux.HandleFunc("GET /_healthz", handleHealth)

	mux.Handle("GET /metrics", atropos.MetricsHandler())
	mux.Handle("/admin/fault", atropos.FaultAdminHandler())
	mux.Handle("/admin/fault/", atropos.FaultAdminHandler()) // subtree: DELETE /admin/fault/{category}
	mux.Handle("/admin/rules", atropos.RulesAdminHandler(eval))
	mountCacheBox(mux, cb, "shippingservice", instanceID)
	mux.Handle("/atropos/health", atropos.HealthHandler())

	srv := &http.Server{
		Addr:    addr,
		Handler: atropos.IngressMiddleware(mux, "shippingservice"),
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Infof("Received signal %v, shutting down", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("HTTP server shutdown error: %v", err)
		}
	}()

	log.Infof("Shipping Service listening on port %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to serve: %v", err)
	}
}

// handleGetQuote produces a shipping quote (cost) in USD.
func handleGetQuote(w http.ResponseWriter, r *http.Request) {
	log.Info("[GetQuote] received request")
	defer log.Info("[GetQuote] completed request")

	var req GetQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Generate a quote based on the total number of items to be shipped.
	count := 0
	for _, item := range req.Items {
		count += int(item.Quantity)
	}
	quote := CreateQuoteFromCount(count)

	resp := GetQuoteResponse{
		CostUsd: Money{
			CurrencyCode: "USD",
			Units:        int64(quote.Dollars),
			Nanos:        int32(quote.Cents * 10000000),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf("[GetQuote] failed to encode response: %v", err)
	}
}

// handleShipOrder mocks that the requested items will be shipped.
// It supplies a tracking ID for notional lookup of shipment delivery status.
func handleShipOrder(w http.ResponseWriter, r *http.Request) {
	log.Info("[ShipOrder] received request")
	defer log.Info("[ShipOrder] completed request")

	var req ShipOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Address == nil {
		http.Error(w, "address is required", http.StatusBadRequest)
		return
	}

	// Create a Tracking ID.
	baseAddress := fmt.Sprintf("%s, %s, %s", req.Address.StreetAddress, req.Address.City, req.Address.State)
	id := CreateTrackingId(baseAddress)

	resp := ShipOrderResponse{
		TrackingID: id,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf("[ShipOrder] failed to encode response: %v", err)
	}
}

// handleHealth returns HTTP 200 to indicate the service is healthy.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "serving"})
}
