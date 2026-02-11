// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetQuote tests the GetQuote HTTP endpoint.
func TestGetQuote(t *testing.T) {
	reqBody := GetQuoteRequest{
		Address: &Address{
			StreetAddress: "Muffin Man",
			City:          "London",
			State:         "",
			Country:       "England",
		},
		Items: []CartItem{
			{ProductID: "23", Quantity: 1},
			{ProductID: "46", Quantity: 3},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/shipping/quote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handleGetQuote(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp GetQuoteResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.CostUsd.CurrencyCode != "USD" {
		t.Errorf("expected currency code USD, got %s", resp.CostUsd.CurrencyCode)
	}
	if resp.CostUsd.Units != 8 || resp.CostUsd.Nanos != 990000000 {
		t.Errorf("expected quote $8.99 (units=8, nanos=990000000), got units=%d, nanos=%d",
			resp.CostUsd.Units, resp.CostUsd.Nanos)
	}
}

// TestShipOrder tests the ShipOrder HTTP endpoint.
func TestShipOrder(t *testing.T) {
	reqBody := ShipOrderRequest{
		Address: &Address{
			StreetAddress: "Muffin Man",
			City:          "London",
			State:         "",
			Country:       "England",
		},
		Items: []CartItem{
			{ProductID: "23", Quantity: 1},
			{ProductID: "46", Quantity: 3},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/shipping/ship", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handleShipOrder(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp ShipOrderResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Tracking IDs follow the pattern: XX-<len>NNN-<len/2>NNNNNNN (18 chars total).
	if len(resp.TrackingID) != 18 {
		t.Errorf("tracking ID is malformed - has %d characters, expected 18: %q",
			len(resp.TrackingID), resp.TrackingID)
	}
}

// TestShipOrderMissingAddress tests that ShipOrder returns 400 when address is nil.
func TestShipOrderMissingAddress(t *testing.T) {
	reqBody := ShipOrderRequest{
		Items: []CartItem{
			{ProductID: "23", Quantity: 1},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/shipping/ship", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handleShipOrder(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing address, got %d", rr.Code)
	}
}

// TestHealth tests the health check endpoint.
func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "serving" {
		t.Errorf("expected status 'serving', got %q", resp["status"])
	}
}
