package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Read configuration from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	catalogAddr := os.Getenv("PRODUCT_CATALOG_SERVICE_ADDR")
	if catalogAddr == "" {
		logger.Error("PRODUCT_CATALOG_SERVICE_ADDR environment variable not set")
		os.Exit(1)
	}
	logger.Info("product catalog address: " + catalogAddr)

	// Initialize OpenTelemetry tracing
	if os.Getenv("ENABLE_TRACING") == "1" {
		tp, err := initTracing(logger)
		if err != nil {
			logger.Warn("failed to initialize tracing", "error", err)
		} else {
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := tp.Shutdown(ctx); err != nil {
					logger.Error("failed to shutdown tracer provider", "error", err)
				}
			}()
		}
	} else {
		logger.Info("Tracing disabled.")
	}

	// Connect to popularity PostgreSQL database
	var popularityPool *pgxpool.Pool
	dbConnStr := os.Getenv("POPULARITY_DB_CONN")
	if dbConnStr != "" {
		dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dbCancel()

		pool, err := pgxpool.New(dbCtx, dbConnStr)
		if err != nil {
			logger.Warn("failed to connect to popularity database, falling back to random", "error", err)
		} else {
			popularityPool = pool
			defer popularityPool.Close()
			logger.Info("connected to popularity database")
		}
	} else {
		logger.Info("POPULARITY_DB_CONN not set, using random recommendations")
	}

	// Create the recommendation service
	service := NewRecommendationService(catalogAddr, popularityPool, logger)

	// Set up HTTP handlers
	mux := http.NewServeMux()

	mux.Handle("POST /recommendations", otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleListRecommendations(w, r, service, logger)
		}),
		"POST /recommendations",
	))

	mux.Handle("GET /_healthz", otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}),
		"GET /_healthz",
	))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Graceful shutdown on SIGINT/SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("RecommendationService started", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("server exited properly")
}

// handleListRecommendations handles POST /recommendations.
// Equivalent to Python: RecommendationService.ListRecommendations
func handleListRecommendations(w http.ResponseWriter, r *http.Request, service *RecommendationService, logger *slog.Logger) {
	var req ListRecommendationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := service.ListRecommendations(r.Context(), &req)
	if err != nil {
		logger.Error("ListRecommendations failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	logger.Info("[Recv ListRecommendations]", "product_ids", resp.ProductIDs)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("failed to encode response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// initTracing sets up the OpenTelemetry trace provider with an OTLP gRPC exporter.
// Equivalent to Python: OTel tracing setup in __main__
func initTracing(logger *slog.Logger) (*sdktrace.TracerProvider, error) {
	collectorAddr := os.Getenv("COLLECTOR_SERVICE_ADDR")
	if collectorAddr == "" {
		collectorAddr = "localhost:4317"
	}

	ctx := context.Background()

	conn, err := grpc.NewClient(collectorAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	logger.Info("Tracing enabled", "collector", collectorAddr)
	return tp, nil
}
