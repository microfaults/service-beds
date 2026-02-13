// Copyright 2026 Google LLC
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
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/profiler"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultPort = "7000"
)

type money struct {
	CurrencyCode string `json:"currency_code"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

type getSupportedCurrenciesResponse struct {
	CurrencyCodes []string `json:"currency_codes"`
}

type currencyConversionRequest struct {
	From   money  `json:"from"`
	ToCode string `json:"to_code"`
}

type healthResponse struct {
	Status string `json:"status"`
}

type service struct {
	log   *logrus.Logger
	rates map[string]float64 // currencyCode -> units-per-EUR
}

func main() {
	log := newLogger()
	ctx := context.Background()

	// Propagate trace context even if tracing is disabled.
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{}))

	if os.Getenv("ENABLE_TRACING") == "1" {
		log.Info("Tracing enabled.")
		if err := initTracing(ctx); err != nil {
			log.Warnf("warn: failed to initialize tracing: %v", err)
		}
	} else {
		log.Info("Tracing disabled.")
	}

	if profilerEnabled() {
		log.Info("Profiling enabled.")
		go initProfiling(log, "currencyservice", "1.0.0")
	} else {
		log.Info("Profiling disabled.")
	}

	port := defaultPort
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	svc := &service{log: log}
	if err := svc.loadRates(); err != nil {
		log.Fatalf("failed to load currency conversion rates: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /_healthz", svc.healthz)
	mux.HandleFunc("GET /currencies", svc.getSupportedCurrencies)
	mux.HandleFunc("POST /convert", svc.convert)

	var handler http.Handler = mux
	handler = otelhttp.NewHandler(handler, "currencyservice")

	addr := ":" + port
	log.Infof("starting HTTP server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func newLogger() *logrus.Logger {
	log := logrus.New()
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
	return log
}

func profilerEnabled() bool {
	// Backwards-compat: legacy Node version used DISABLE_PROFILER=1.
	if os.Getenv("DISABLE_PROFILER") == "1" {
		return false
	}
	return os.Getenv("ENABLE_PROFILER") == "1"
}

func initProfiling(log logrus.FieldLogger, serviceName, version string) {
	for i := 1; i <= 3; i++ {
		log = log.WithField("retry", i)
		if err := profiler.Start(profiler.Config{
			Service:        serviceName,
			ServiceVersion: version,
		}); err != nil {
			log.Warnf("warn: failed to start profiler: %+v", err)
		} else {
			log.Info("started profiler")
			return
		}
		d := time.Second * 10 * time.Duration(i)
		log.Debugf("sleeping %v to retry initializing profiler", d)
		time.Sleep(d)
	}
	log.Warn("warning: could not initialize profiler after retrying, giving up")
}

func initTracing(ctx context.Context) error {
	collector := os.Getenv("COLLECTOR_SERVICE_ADDR")
	if collector == "" {
		return fmt.Errorf("COLLECTOR_SERVICE_ADDR not set")
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(collector,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	return nil
}

func (s *service) loadRates() error {
	// Keep path compatible with the existing directory layout.
	path := filepath.Join(".", "data", "currency_conversion.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// File uses string values, e.g. "USD": "1.1305".
	raw := map[string]string{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	out := make(map[string]float64, len(raw))
	for code, v := range raw {
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return fmt.Errorf("parse rate %s=%q: %w", code, v, err)
		}
		if f <= 0 {
			return fmt.Errorf("invalid rate %s=%v", code, f)
		}
		out[code] = f
	}
	s.rates = out
	return nil
}

func (s *service) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "SERVING"})
}

func (s *service) getSupportedCurrencies(w http.ResponseWriter, _ *http.Request) {
	codes := make([]string, 0, len(s.rates))
	for code := range s.rates {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	writeJSON(w, http.StatusOK, getSupportedCurrenciesResponse{CurrencyCodes: codes})
}

func (s *service) convert(w http.ResponseWriter, r *http.Request) {
	var req currencyConversionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	fromRate, ok := s.rates[req.From.CurrencyCode]
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported from currency_code %q", req.From.CurrencyCode))
		return
	}
	toRate, ok := s.rates[req.ToCode]
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported to_code %q", req.ToCode))
		return
	}

	// Convert: from_currency -> EUR -> to_currency.
	fromAmount := float64(req.From.Units) + float64(req.From.Nanos)/1e9
	eur := fromAmount / fromRate
	// Replicate Node.js behavior: round to nearest nano (9 decimal places).
	eur = round(eur*1e9) / 1e9
	toAmount := eur * toRate
	out := moneyFromFloat(toAmount, req.ToCode)

	s.log.WithFields(logrus.Fields{
		"from_currency": req.From.CurrencyCode,
		"to_currency":   req.ToCode,
	}).Info("conversion request successful")

	writeJSON(w, http.StatusOK, out)
}

func moneyFromFloat(v float64, code string) money {
	units := int64(trunc(v))
	nanos := int32(round((v - float64(units)) * 1e9))

	// Carry if rounding produced exactly 1e9 nanos.
	if nanos >= 1_000_000_000 {
		units++
		nanos -= 1_000_000_000
	}
	if nanos <= -1_000_000_000 {
		units--
		nanos += 1_000_000_000
	}
	return money{
		CurrencyCode: code,
		Units:        units,
		Nanos:        nanos,
	}
}

func trunc(v float64) float64 {
	if v >= 0 {
		return float64(int64(v))
	}
	return float64(int64(v)) // int64 truncates toward zero
}

func round(v float64) float64 {
	if v >= 0 {
		return float64(int64(v + 0.5))
	}
	return float64(int64(v - 0.5))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
