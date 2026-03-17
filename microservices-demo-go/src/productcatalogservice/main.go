package main

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/microfaults/atropos-go"

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
	shutdown, err := atropos.Init(ctx,
		atropos.WithServiceName("productcatalogservice"),
		atropos.WithServiceVersion("0.1.0"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer shutdown(ctx)

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

	mux.Handle("GET /metrics", atropos.MetricsHandler())
	mux.Handle("/admin/fault", atropos.FaultAdminHandler())

	log.Infof("starting http server at :%s", port)
	if err := http.ListenAndServe(":"+port, atropos.IngressMiddleware(mux, "productcatalogservice")); err != nil {
		log.Fatal(err)
	}
}
