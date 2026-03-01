package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/cartstore"
	"github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/model"
	"github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/service"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

func main() {
	ctx := context.Background()

otel.SetTextMapPropagator(
  propagation.NewCompositeTextMapPropagator(
    propagation.TraceContext{}, propagation.Baggage{},
  ),
)
	port := os.Getenv("PORT")
	if port == "" {
		port = "7070" // Default to original port if not specified
	}
	// Allow overriding HTTP port specifically if needed, otherwise use PORT
	if httpPort := os.Getenv("HTTP_PORT"); httpPort != "" {
		port = httpPort
	}
	if port == "" {
		port = "8080"
	}
	if os.Getenv("ENABLE_TRACING") == "1" {
  if err := initTracing(ctx); err != nil {
    log.Printf("failed to initialize tracing: %v", err)
  }
}
	var store cartstore.CartStore
	redisAddr := os.Getenv("REDIS_ADDR")
	alloyDBPrimaryIP := os.Getenv("ALLOYDB_PRIMARY_IP")

	if redisAddr != "" {
		log.Printf("Using RedisCartStore with address: %s", redisAddr)
		store = cartstore.NewRedisCartStore(redisAddr)
	} else if alloyDBPrimaryIP != "" {
		// AlloyDB Configuration
		log.Printf("Using AlloyDBCartStore with IP: %s", alloyDBPrimaryIP)

		user := os.Getenv("ALLOYDB_USER")
		if user == "" {
			user = "postgres"
		}

		password := os.Getenv("ALLOYDB_PASSWORD") // TODO: Integrate Secret Manager if needed
		dbname := os.Getenv("ALLOYDB_DATABASE_NAME")
		if dbname == "" {
			dbname = "postgres"
		}

		tableName := os.Getenv("ALLOYDB_TABLE_NAME")
		if tableName == "" {
			tableName = "carts"
		}

		// TODO: Context cancellation for connection?
		// For main implementation, we can block or use background
		pool, err := cartstore.ConnectAlloyDB(context.Background(), alloyDBPrimaryIP, user, password, dbname)
		if err != nil {
			log.Fatalf("Failed to connect to AlloyDB: %v", err)
		}

		store = cartstore.NewAlloyDBCartStore(context.Background(), pool, tableName)
	} else {
		log.Println("REDIS_ADDR and ALLOYDB_PRIMARY_IP not set. Using MemoryCartStore.")
		store = cartstore.NewMemoryCartStore()
	}

	svc := service.NewCartService(store)
	mux := http.NewServeMux()

	// POST /cart/{user_id}/items
	mux.HandleFunc("POST /cart/{user_id}/items", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("user_id")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		input := &model.CartItem{}
		if err := json.Unmarshal(body, input); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse JSON: %v. Expected {'product_id': '...', 'quantity': N}", err), http.StatusBadRequest)
			return
		}

		if input.ProductID == "" {
			http.Error(w, "Missing 'product_id' field in JSON", http.StatusBadRequest)
			return
		}

		err = svc.AddItem(r.Context(), userID, input.ProductID, input.Quantity)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to add item: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	})

	// GET /cart/{user_id}
	mux.HandleFunc("GET /cart/{user_id}", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("user_id")

		cart, err := svc.GetCart(r.Context(), userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get cart: %v", err), http.StatusInternalServerError)
			return
		}

		respBytes, err := json.Marshal(cart)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(respBytes)
	})

	// DELETE /cart/{user_id}
	mux.HandleFunc("DELETE /cart/{user_id}", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("user_id")

		err := svc.EmptyCart(r.Context(), userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to empty cart: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	})

	// GET /_healthz
	mux.HandleFunc("GET /_healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := otelhttp.NewHandler(mux, "cartservice")

log.Printf("HTTP server listening on :%s", port)
if err := http.ListenAndServe(fmt.Sprintf(":%s", port), handler); err != nil {
    log.Fatalf("failed to serve HTTP: %v", err)
}
}


func initTracing(ctx context.Context) (*sdktrace.TracerProvider, func(), error) {
    collector := os.Getenv("COLLECTOR_SERVICE_ADDR")
    if collector == "" {
        return nil, nil, fmt.Errorf("COLLECTOR_SERVICE_ADDR not set")
    }

    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    exporter, err := otlptracegrpc.New(
        ctx,
        otlptracegrpc.WithEndpoint(collector),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithSampler(sdktrace.AlwaysSample()),
    )
    otel.SetTracerProvider(tp)

    cleanup := func() {
        _ = tp.Shutdown(context.Background())
    }

    return tp, cleanup, nil
}