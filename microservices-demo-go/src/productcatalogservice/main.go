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
	shutdown, err := atropos.Init(ctx,
		atropos.WithServiceName("productcatalogservice"),
		atropos.WithServiceVersion("0.1.0"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer shutdown(ctx)

	eval := atropos.NewStaticEvaluator()

	var cbPush *atropos.CachePushClient
	cbCfg := atropos.CacheBoxConfig{
		Store: atropos.NewCacheBoxMemStore(1000),
	}
	if manteionURL := os.Getenv("MANTEION_URL"); manteionURL != "" {
		cbPush = atropos.NewCachePushClient(atropos.CachePushConfig{
			BaseURL:  manteionURL,
			Service:  "productcatalogservice",
			Instance: os.Getenv("HOSTNAME"),
		})
		cbCfg.Push = cbPush.PushFunc()
	}

	cb := atropos.NewCacheBox(cbCfg)
	atropos.Configure(
		atropos.WithEvaluator(eval),
		atropos.WithCacheBoxCoordinator(cb),
	)

	atropos.RegisterRoutes(
		atropos.Route{Method: "GET", Path: "/products", Description: "List all products in the catalog"},
		atropos.Route{Method: "GET", Path: "/products/{id}", Description: "Fetch a single product by ID"},
		atropos.Route{Method: "POST", Path: "/products/batch", Description: "Fetch multiple products by ID list"},
		atropos.Route{Method: "GET", Path: "/products/search", Description: "Search products by name or description (query: q)"},
	)

	mc, err := atropos.ConnectManteion(ctx, "productcatalogservice",
		atropos.WithApplyTargets(atropos.ApplyTargets{Evaluator: eval, CacheBox: cb}),
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
	mux.Handle("/admin/rules", atropos.RulesAdminHandler(eval))
	mux.Handle("/admin/cachebox", atropos.CacheBoxAdminHandler(cb))
	mux.Handle("/atropos/health", atropos.HealthHandler())

	log.Infof("starting http server at :%s", port)
	if err := http.ListenAndServe(":"+port, atropos.IngressMiddleware(mux, "productcatalogservice")); err != nil {
		log.Fatal(err)
	}
}
