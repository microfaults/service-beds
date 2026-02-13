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
)

func main() {
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

		// Input expectation: {"item": {"product_id": "...", "quantity": 1}}
		// OR just straight item?
		// PROTOS had AddItemRequest { UserID, Item { ProductID, Quantity } }
		// HTTP Gateway mapped body to *, so client sent {"item": {...}} or just { ... } fields?
		// Let's assume a simple JSON object: {"product_id": "...", "quantity": 1}
		// OR to match previous verification: {"item": {"product_id": "...", "quantity": 1}}
		// The simplest pure-HTTP design is: {"product_id": "...", "quantity": 1}

		// Let's support the simple flat structure for the "Pure HTTP" refactor as it's cleaner.
		// Struct:
		type AddItemRequest struct {
			ProductID string `json:"product_id"`
			Quantity  int32  `json:"quantity"`
			// Optional wrapper support if needed, but let's stick to flat for now unless verification fails
		}
		// UPDATE: The user asked for "structs interpreted from protos".
		// Proto AddItemRequest had `CartItem item = 2`.
		// So the JSON likely was `{"item": {"product_id": "...", "quantity": ...}}`.
		// To maintain compatibility with the previous curl commands I gave:
		// `curl -X POST -d '{"item": {"product_id": "test-product", "quantity": 1}}'`
		// I should probably support that structure.

		type AddItemWrapper struct {
			Item *model.CartItem `json:"item"`
		}

		input := &AddItemWrapper{}
		if err := json.Unmarshal(body, input); err != nil {
			// Try flat Unmarshal as fallback? Or strict?
			// Let's stick to the wrapper to match the previous walkthrough's curl
			http.Error(w, fmt.Sprintf("Failed to parse JSON: %v. Expected {'item': {'product_id': '...', 'quantity': N}}", err), http.StatusBadRequest)
			return
		}

		if input.Item == nil {
			http.Error(w, "Missing 'item' field in JSON", http.StatusBadRequest)
			return
		}

		err = svc.AddItem(r.Context(), userID, input.Item.ProductID, input.Item.Quantity)
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

	log.Printf("HTTP server listening on :%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), mux); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
