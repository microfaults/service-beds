package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/microservices-demo-go/src/frontend/clients"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{}))

	baseUrl = os.Getenv("BASE_URL")
	
	tp, err := initTracing(log, ctx)
	if err != nil {
		log.Fatalf("failed to initialize tracing: %v", err)
	}
	if tp != nil {
    defer func() {
        if err := tp.Shutdown(context.Background()); err != nil {
            log.Errorf("failed to shutdown TracerProvider: %v", err)
        }
    }()
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
	handler = &logHandler{log: log, next: handler}     // add logging
	handler = ensureSessionID(handler)                 // add session ID
	handler = otelhttp.NewHandler(handler, "frontend") // add OTel tracing

	log.Infof("starting server on %s:%s", addr, srvPort)
	log.Fatal(http.ListenAndServe(addr+":"+srvPort, handler))
}

func initTracing(log logrus.FieldLogger, ctx context.Context) (*sdktrace.TracerProvider, error) {
	collectorAddr := os.Getenv("COLLECTOR_SERVICE_ADDR")
	if collectorAddr == "" {
		log.Info("COLLECTOR_SERVICE_ADDR not set, tracing exporter not initialized")
		return nil, nil
	}

	// Create exporter for OTLP
	// Insecure for demo purposes
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(collectorAddr),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Warnf("warn: Failed to create trace exporter: %v", err)
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(tp)

	return tp, nil
}

func mustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		panic(fmt.Sprintf("environment variable %q not set", envKey))
	}
	*target = v
}
