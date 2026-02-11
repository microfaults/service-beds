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

// Address represents a shipping address.
type Address struct {
	StreetAddress string `json:"street_address"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
	ZipCode       int32  `json:"zip_code"`
}

// CartItem represents an item in the shopping cart.
type CartItem struct {
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

// Money represents an amount of money with its currency type.
type Money struct {
	CurrencyCode string `json:"currency_code"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

// GetQuoteRequest is the request payload for the GetQuote endpoint.
type GetQuoteRequest struct {
	Address *Address   `json:"address"`
	Items   []CartItem `json:"items"`
}

// GetQuoteResponse is the response payload for the GetQuote endpoint.
type GetQuoteResponse struct {
	CostUsd Money `json:"cost_usd"`
}

// ShipOrderRequest is the request payload for the ShipOrder endpoint.
type ShipOrderRequest struct {
	Address *Address   `json:"address"`
	Items   []CartItem `json:"items"`
}

// ShipOrderResponse is the response payload for the ShipOrder endpoint.
type ShipOrderResponse struct {
	TrackingID string `json:"tracking_id"`
}
