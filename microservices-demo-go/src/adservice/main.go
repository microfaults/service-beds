package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/microfaults/atropos-go"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9555"
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx := context.Background()
	shutdown, err := atropos.Init(ctx,
		atropos.WithServiceName("adservice"),
		atropos.WithServiceVersion("0.1.0"),
	)
	if err != nil {
		logger.Error("failed to init atropos", "error", err)
		os.Exit(1)
	}
	defer shutdown(ctx)

	eval := atropos.NewStaticEvaluator()
	cb := atropos.NewCacheBox(atropos.CacheBoxConfig{
		Store: atropos.NewCacheBoxMemStore(1000),
	})
	atropos.Configure(
		atropos.WithEvaluator(eval),
		atropos.WithCacheBoxCoordinator(cb),
	)

	mc, err := atropos.ConnectManteion(ctx, "adservice",
		atropos.WithApplyTargets(atropos.ApplyTargets{Evaluator: eval, CacheBox: cb}),
	)
	if err != nil {
		logger.Warn("manteion connection failed, running offline", "error", err)
	}
	if mc != nil {
		defer mc.Close(ctx)
	}

	service := NewService()

	mux := http.NewServeMux()
	mux.HandleFunc("/ads", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req AdRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		logger.Info("received ad request", "context_keys", req.ContextKeys)

		ads := service.GetAdsByCategory(req.ContextKeys)
		resp := AdResponse{Ads: ads}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			logger.Error("failed to encode response", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	mux.HandleFunc("/_healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.Handle("GET /metrics", atropos.MetricsHandler())
	mux.Handle("/admin/fault", atropos.FaultAdminHandler())
	mux.Handle("/admin/rules", atropos.RulesAdminHandler(eval))
	mux.Handle("/admin/cachebox", atropos.CacheBoxAdminHandler(cb))
	mux.Handle("/atropos/health", atropos.HealthHandler())

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: atropos.IngressMiddleware(mux, "adservice"),
	}

	// Channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run server in a goroutine so it doesn't block the main thread
	go func() {
		logger.Info("AdService started", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	// Block until we receive a signal
	<-stop
	logger.Info("shutting down server...")

	// Create a deadline to wait for current operations to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("server exited properly")
}
