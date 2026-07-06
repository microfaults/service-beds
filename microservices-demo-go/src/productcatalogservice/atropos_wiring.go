// atropos_wiring.go is the canonical atropos cache-box fidelity wiring, shared
// VERBATIM across the ServeMux demo-go services so the fidelity pipeline reads
// the same everywhere. Edit it in one service and re-copy to all; do not let the
// copies diverge.
package main

import (
	"net/http"
	"os"

	"git.ucsc.edu/microfaults/atropos-go"
)

// resolveInstanceID picks the id that MUST be identical across the three places
// manteion correlates this SDK instance -- the register call (WithInstanceID),
// the cache-push client (ingest envelopes + W3 drain reports), and the fidelity
// handler. Precedence: MANTEION_INSTANCE_ID > hostname (the pod name in k8s) >
// service name.
func resolveInstanceID(service string) string {
	if id := os.Getenv("MANTEION_INSTANCE_ID"); id != "" {
		return id
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return service
}

// newCachePush builds the manteion cache-push client, or returns nil when
// MANTEION_URL is unset (offline mode). After building the CacheBox, bind it
// with cbPush.BindFidelity(cb.Fidelity()) before any traffic flows so the drain
// report's push-side counts are real.
func newCachePush(service, instanceID string) *atropos.CachePushClient {
	manteionURL := os.Getenv("MANTEION_URL")
	if manteionURL == "" {
		return nil
	}
	return atropos.NewCachePushClient(atropos.CachePushConfig{
		BaseURL:  manteionURL,
		Service:  service,
		Instance: instanceID,
	})
}

// mountCacheBox mounts the manteion cache-box control surface on a net/http
// ServeMux at the exact paths manteion addresses (these services carry no
// BASE_URL prefix). The admin handler is mounted at BOTH the bare path (GET
// stats, DELETE thaw) and the subtree (POST /admin/cachebox/delay): a ServeMux
// subtree pattern alone would 301-redirect the bare path and drop the
// freeze/thaw verbs.
func mountCacheBox(mux *http.ServeMux, cb *atropos.CacheBox, service, instanceID string) {
	mux.Handle("/admin/cachebox", atropos.CacheBoxAdminHandler(cb))
	mux.Handle("/admin/cachebox/", atropos.CacheBoxAdminHandler(cb))
	mux.Handle("/cachebox/preload/", atropos.CacheBoxPreloadHandler(cb))
	mux.Handle("GET /cachebox/fidelity", atropos.CacheBoxFidelityHandler(cb, service, instanceID))
}
