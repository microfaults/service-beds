package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"git.ucsc.edu/microfaults/atropos-go"

	"github.com/GoogleCloudPlatform/microservices-demo-go/src/shoppingassistantservice/internal/db"
	"github.com/GoogleCloudPlatform/microservices-demo-go/src/shoppingassistantservice/internal/llm"
)

var (
	projectID         = os.Getenv("PROJECT_ID")
	region            = os.Getenv("REGION")
	alloyDBCluster    = os.Getenv("ALLOYDB_CLUSTER_NAME")
	alloyDBInstance   = os.Getenv("ALLOYDB_INSTANCE_NAME")
	alloyDBDatabase   = os.Getenv("ALLOYDB_DATABASE_NAME")
	alloyDBSecretName = os.Getenv("ALLOYDB_SECRET_NAME")
	alloyDBTable      = os.Getenv("ALLOYDB_TABLE_NAME") // Used in handler
)

func main() {
	if projectID == "" {
		log.Fatal("PROJECT_ID environment variable is required")
	}

	ctx := context.Background()

	shutdown, err := atropos.Init(ctx,
		atropos.WithServiceName("shoppingassistantservice"),
		atropos.WithServiceVersion("0.1.0"),
	)
	if err != nil {
		log.Fatalf("failed to init atropos: %v", err)
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
			Service:  "shoppingassistantservice",
			Instance: os.Getenv("HOSTNAME"),
		})
		cbCfg.Push = cbPush.PushFunc()
	}

	cb := atropos.NewCacheBox(cbCfg)
	atropos.Configure(
		atropos.WithEvaluator(eval),
		atropos.WithCacheBoxCoordinator(cb),
	)

	mc, err := atropos.ConnectManteion(ctx, "shoppingassistantservice",
		atropos.WithApplyTargets(atropos.ApplyTargets{Evaluator: eval, CacheBox: cb}),
	)
	if err != nil {
		log.Printf("manteion connection failed, running offline: %v", err)
	}
	if mc != nil {
		defer mc.Close(ctx)
	}
	if cbPush != nil {
		defer cbPush.Stop()
	}

	// 2. Initialize Product Store
	var productStore db.ProductStore

	dbBackend := os.Getenv("DB_BACKEND")
	if dbBackend == "alloydb" {
		productStore, err = db.NewAlloyDBStore(ctx, projectID, region, alloyDBCluster, alloyDBInstance, alloyDBDatabase, alloyDBSecretName, alloyDBTable)
		if err != nil {
			log.Fatalf("failed to create alloydb store: %v", err)
		}
	} else if dbBackend == "postgres" {
		// Expecting standard PG environment variables or a DSN
		// For simplicity, let's assume a DSN env var "DB_DSN" or build it from standard PG vars.
		// Let's use DB_DSN for flexibility.
		dsn := os.Getenv("DB_DSN")
		if dsn == "" {
			// Fallback/Default for local development
			dsn = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
		}
		productStore, err = db.NewPostgresStore(ctx, dsn, alloyDBTable)
		if err != nil {
			log.Fatalf("failed to create postgres store: %v", err)
		}
	} else {
		log.Println("Using Mock Product Store")
		productStore = db.NewMockStore()
	}

	// 3. Initialize LLM Client
	// For now, we use the MockClient by default or based on env var.
	// In future, we can switch based on LLM_BACKEND env var.
	var llmClient llm.Client
	if os.Getenv("LLM_BACKEND") == "google" {
		var err error
		llmClient, err = llm.NewGoogleClient(ctx, projectID, region)
		if err != nil {
			log.Fatalf("failed to create google llm client: %v", err)
		}
	} else {
		log.Println("Using Mock LLM Client")
		llmClient = llm.NewMockClient()
	}

	// 4. Setup HTTP Server
	h := &Handler{
		productStore: productStore,
		llmClient:    llmClient,
		projectID:    projectID,
		location:     region,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.talkToGemini)
	mux.HandleFunc("/_healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.Handle("GET /metrics", atropos.MetricsHandler())
	mux.Handle("/admin/fault", atropos.FaultAdminHandler())
	mux.Handle("/admin/rules", atropos.RulesAdminHandler(eval))
	mux.Handle("/admin/cachebox", atropos.CacheBoxAdminHandler(cb))
	mux.Handle("/atropos/health", atropos.HealthHandler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, atropos.IngressMiddleware(mux, "shoppingassistantservice")); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
