package main

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

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

	svc := &productCatalog{}
	handler := &ProductHandler{service: svc}

	if err := watchProductsFile(context.Background()); err != nil {
		log.Warnf("failed to start file watcher: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /products", handler.ListProducts)
	mux.HandleFunc("GET /products/{id}", handler.GetProduct)
	mux.HandleFunc("GET /products/search", handler.SearchProducts)
	mux.HandleFunc("GET /_healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Infof("starting http server at :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
