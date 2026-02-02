package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Logger for email client
var clientLogger = GetJSONLogger("emailservice-client")

// SendConfirmationEmail sends an order confirmation email via HTTP POST request
// This is equivalent to the Python send_confirmation_email function
func SendConfirmationEmail(email string, order *OrderResult) error {
	// Create the request payload
	request := SendOrderConfirmationRequest{
		Email: email,
		Order: order,
	}

	// Marshal request to JSON
	jsonBody, err := json.Marshal(request)
	if err != nil {
		clientLogger.Error(fmt.Sprintf("Failed to marshal request: %s", err.Error()))
		return err
	}

	// Send HTTP POST request to the email service
	// Equivalent to Python: grpc.insecure_channel('[::]:8080')
	url := "http://localhost:8080/send-order-confirmation"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		clientLogger.Error(fmt.Sprintf("HTTP request failed: %s", err.Error()))
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Request failed with status: %d %s", resp.StatusCode, resp.Status)
		clientLogger.Error(errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	clientLogger.Info("Request sent.")
	return nil
}

// RunEmailClient is the main entry point when running as a standalone client
// Equivalent to Python: if __name__ == '__main__'
func RunEmailClient() {
	clientLogger.Info("Client for email service.")
}