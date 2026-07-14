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

	"git.ucsc.edu/microfaults/atropos-go"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9555"
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx := context.Background()

	service := NewService()

	mux := http.NewServeMux()
	mux.HandleFunc("/ads", func(w http.ResponseWriter, r *http.Request) {
		// GetAds is the REST projection of the upstream gRPC
		// `AdService.GetAds(AdRequest{context_keys})` read RPC. Reads use GET
		// with repeated `?context_keys=` query parameters; the frontend client
		// in src/frontend/clients/clients.go:GetAds is the canonical caller.
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		contextKeys := r.URL.Query()["context_keys"]

		logger.Info("received ad request", "context_keys", contextKeys)

		ads := service.GetAdsByCategory(contextKeys)
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

	h, shutdown, err := atropos.Serve(ctx, atropos.Config{
		Service: "adservice",
		Version: "0.1.0",
		Routes: []atropos.Route{
			{Method: "GET", Path: "/ads", Description: "List ad candidates for the given context keys (query: context_keys)"},
		},
		Handler: mux,
	})
	if err != nil {
		logger.Error("failed to init atropos", "error", err)
		os.Exit(1)
	}
	defer shutdown(ctx)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: h,
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
