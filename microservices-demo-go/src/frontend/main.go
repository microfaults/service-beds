package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"atropos-go"

	"github.com/GoogleCloudPlatform/microservices-demo-go/src/frontend/clients"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

const (
	port            = "8080"
	defaultCurrency = "USD"
	cookieMaxAge    = 60 * 60 * 48

	cookiePrefix    = "shop_"
	cookieSessionID = cookiePrefix + "session-id"
	cookieCurrency  = cookiePrefix + "currency"
)

var (
	whitelistedCurrencies = map[string]bool{
		"USD": true,
		"EUR": true,
		"CAD": true,
		"JPY": true,
		"GBP": true,
		"TRY": true,
	}

	baseUrl = ""
)

type ctxKeySessionID struct{}

type frontendServer struct {
	productCatalogSvc clients.ProductCatalogClient
	currencySvc       clients.CurrencyClient
	cartSvc           clients.CartClient
	recommendationSvc clients.RecommendationClient
	checkoutSvc       clients.CheckoutClient
	shippingSvc       clients.ShippingClient
	adSvc             clients.AdClient

	shoppingAssistantSvcAddr string // kept as address for now, or use client if needed
}

func main() {

	ctx := context.Background()
	log := logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout

	baseUrl = os.Getenv("BASE_URL")

	shutdown, err := atropos.Init(ctx,
		atropos.WithServiceName("frontend"),
		atropos.WithServiceVersion("0.1.0"),
	)
	if err != nil {
		log.Warnf("failed to init atropos: %v", err)
	}
	if shutdown != nil {
		defer shutdown(ctx)
	}
	srvPort := port
	if os.Getenv("PORT") != "" {
		srvPort = os.Getenv("PORT")
	}
	addr := os.Getenv("LISTEN_ADDR")

	var (
		productCatalogSvcAddr    string
		currencySvcAddr          string
		cartSvcAddr              string
		recommendationSvcAddr    string
		checkoutSvcAddr          string
		shippingSvcAddr          string
		adSvcAddr                string
		shoppingAssistantSvcAddr string
	)

	mustMapEnv(&productCatalogSvcAddr, "PRODUCT_CATALOG_SERVICE_ADDR")
	mustMapEnv(&currencySvcAddr, "CURRENCY_SERVICE_ADDR")
	mustMapEnv(&cartSvcAddr, "CART_SERVICE_ADDR")
	mustMapEnv(&recommendationSvcAddr, "RECOMMENDATION_SERVICE_ADDR")
	mustMapEnv(&checkoutSvcAddr, "CHECKOUT_SERVICE_ADDR")
	mustMapEnv(&shippingSvcAddr, "SHIPPING_SERVICE_ADDR")
	mustMapEnv(&adSvcAddr, "AD_SERVICE_ADDR")
	mustMapEnv(&shoppingAssistantSvcAddr, "SHOPPING_ASSISTANT_SERVICE_ADDR")

	// Initialize Clients
	pcClient := clients.NewProductCatalogClient(productCatalogSvcAddr)

	svc := &frontendServer{
		productCatalogSvc:        pcClient,
		currencySvc:              clients.NewCurrencyClient(currencySvcAddr),
		cartSvc:                  clients.NewCartClient(cartSvcAddr),
		recommendationSvc:        clients.NewRecommendationClient(recommendationSvcAddr, pcClient),
		checkoutSvc:              clients.NewCheckoutClient(checkoutSvcAddr),
		shippingSvc:              clients.NewShippingClient(shippingSvcAddr),
		adSvc:                    clients.NewAdClient(adSvcAddr),
		shoppingAssistantSvcAddr: shoppingAssistantSvcAddr,
	}

	r := mux.NewRouter()
	r.HandleFunc(baseUrl+"/", svc.homeHandler).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc(baseUrl+"/product/{id}", svc.productHandler).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc(baseUrl+"/cart", svc.viewCartHandler).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc(baseUrl+"/cart", svc.addToCartHandler).Methods(http.MethodPost)
	r.HandleFunc(baseUrl+"/cart/empty", svc.emptyCartHandler).Methods(http.MethodPost)
	r.HandleFunc(baseUrl+"/setCurrency", svc.setCurrencyHandler).Methods(http.MethodPost)
	r.HandleFunc(baseUrl+"/logout", svc.logoutHandler).Methods(http.MethodGet)
	r.HandleFunc(baseUrl+"/cart/checkout", svc.placeOrderHandler).Methods(http.MethodPost)
	r.HandleFunc(baseUrl+"/assistant", svc.assistantHandler).Methods(http.MethodGet)
	r.PathPrefix(baseUrl + "/static/").Handler(http.StripPrefix(baseUrl+"/static/", http.FileServer(http.Dir("./static/"))))
	r.HandleFunc(baseUrl+"/robots.txt", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "User-agent: *\nDisallow: /") })
	r.HandleFunc(baseUrl+"/_healthz", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "ok") })
	r.HandleFunc(baseUrl+"/product-meta/{ids}", svc.getProductByID).Methods(http.MethodGet)
	r.HandleFunc(baseUrl+"/bot", svc.chatBotHandler).Methods(http.MethodPost)

	var handler http.Handler = r
	handler = &logHandler{log: log, next: handler} // add logging
	handler = ensureSessionID(handler)             // add session ID
	handler = atropos.IngressMiddleware(handler, "frontend")

	log.Infof("starting server on %s:%s", addr, srvPort)
	log.Fatal(http.ListenAndServe(addr+":"+srvPort, handler))
}

func mustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		panic(fmt.Sprintf("environment variable %q not set", envKey))
	}
	*target = v
}
