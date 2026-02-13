package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)

// Logger for email server
var logger = GetJSONLogger("emailservice-server")

// Loaded confirmation email template
var emailTemplate *template.Template

// init loads the email template at startup
// Equivalent to Python: template = env.get_template('confirmation.html')
func init() {
	templatePath := filepath.Join("templates", "confirmation.html")
	var err error
	emailTemplate, err = template.New("confirmation.html").Funcs(template.FuncMap{
		"formatCents": func(nanos int32) string {
			return fmt.Sprintf("%02d", nanos/10000000)
		},
	}).ParseFiles(templatePath)
	if err != nil {
		logger.Warning(fmt.Sprintf("Failed to load email template: %s", err.Error()))
	}
}

// handleSendOrderConfirmation handles POST /send-order-confirmation
// Equivalent to Python: DummyEmailService.SendOrderConfirmation
func handleSendOrderConfirmation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request SendOrderConfirmationRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		logger.Error(fmt.Sprintf("Failed to decode request: %s", err.Error()))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Log the request (dummy mode - equivalent to Python DummyEmailService)
	logger.Info(fmt.Sprintf("A request to send order confirmation email to %s has been received.", request.Email))

	// In a real implementation, we would render the template and send the email
	// For now, we just log the request (dummy mode)
	if emailTemplate != nil && request.Order != nil {
		// Template is available, we could render it here
		logger.Info("Request sent.")
	}

	// Return empty response (equivalent to Python: return demo_pb2.Empty())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct{}{})
}

// handleHealth handles GET /health
// Equivalent to Python: BaseEmailService.Check (health check)
func handleHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "SERVING",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// start initializes and starts the HTTP server
// Equivalent to Python: def start(dummy_mode)
func start(dummyMode bool) {
	if dummyMode {
		logger.Info("Starting the email service in dummy mode.")
	}

	// Set up HTTP handlers
	http.HandleFunc("/send-order-confirmation", handleSendOrderConfirmation)
	http.HandleFunc("/_healthz", handleHealth)

	// Get port from environment (equivalent to Python: os.environ.get('PORT', "8080"))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info(fmt.Sprintf("listening on port: %s", port))

	// Start HTTP server (equivalent to Python: server.start())
	addr := fmt.Sprintf(":%s", port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Error(fmt.Sprintf("Server failed: %s", err.Error()))
	}
}

// main is the entry point
// Equivalent to Python: if __name__ == '__main__'
func main() {
	logger.Info("starting the email service in dummy mode.")

	// Profiler disabled by default (equivalent to Python profiler logic)
	if os.Getenv("DISABLE_PROFILER") == "" {
		logger.Info("Profiler disabled.")
	}

	// Tracing disabled by default (equivalent to Python tracing logic)
	if os.Getenv("ENABLE_TRACING") != "1" {
		logger.Info("Tracing disabled.")
	}

	// Start in dummy mode (equivalent to Python: start(dummy_mode = True))
	start(true)
}
