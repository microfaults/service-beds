package main

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"git.ucsc.edu/microfaults/atropos-go"

	"github.com/sirupsen/logrus"
)

var (
	log           *logrus.Logger
	catalogMutex  *sync.Mutex
	extraLatency  time.Duration
	reloadCatalog bool
)

func init() {
	log = logrus.New()
	log.Out = os.Stdout
	catalogMutex = &sync.Mutex{}
}

func main() {
	if s := os.Getenv("EXTRA_LATENCY"); s != "" {
		v, err := time.ParseDuration(s)
		if err == nil {
			extraLatency = v
			log.Infof("extra latency enabled (duration: %v)", extraLatency)
		}
	}

	port := "3550"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	ctx := context.Background()

	svc := &productCatalog{}
	handler := &ProductHandler{service: svc}

	if err := watchProductsFile(ctx); err != nil {
		log.Warnf("failed to start file watcher: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /products", handler.ListProducts)
	mux.HandleFunc("POST /products/batch", handler.GetProducts)
	mux.HandleFunc("GET /products/{id}", handler.GetProduct)
	mux.HandleFunc("GET /products/search", handler.SearchProducts)
	mux.HandleFunc("GET /_healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h, shutdown, err := atropos.Serve(ctx, atropos.Config{
		Service: "productcatalogservice",
		Version: "0.1.0",
		Routes: []atropos.Route{
			{Method: "GET", Path: "/products", Description: "List all products in the catalog"},
			{Method: "GET", Path: "/products/{id}", Description: "Fetch a single product by ID"},
			{Method: "POST", Path: "/products/batch", Description: "Fetch multiple products by ID list"},
			{Method: "GET", Path: "/products/search", Description: "Search products by name or description (query: q)"},
		},
		Handler: mux,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer shutdown(ctx)

	log.Infof("starting http server at :%s", port)
	if err := http.ListenAndServe(":"+port, h); err != nil {
		log.Fatal(err)
	}
}
