// Copyright 2026 Google LLC
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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"atropos-go"

	"cloud.google.com/go/profiler"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	defaultPort = "7000"
)

type money struct {
	CurrencyCode string `json:"currency_code"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

type creditCardInfo struct {
	CreditCardNumber          string `json:"credit_card_number"`
	CreditCardCVV             int32  `json:"credit_card_cvv"`
	CreditCardExpirationYear  int32  `json:"credit_card_expiration_year"`
	CreditCardExpirationMonth int32  `json:"credit_card_expiration_month"`
}

type chargeRequest struct {
	Amount     money          `json:"amount"`
	CreditCard creditCardInfo `json:"credit_card"`
}

type chargeResponse struct {
	TransactionID string `json:"transaction_id"`
}

type healthResponse struct {
	Status string `json:"status"`
}

type service struct {
	log *logrus.Logger
}

func main() {
	log := newLogger()
	ctx := context.Background()

	shutdown, err := atropos.Init(ctx,
		atropos.WithServiceName("paymentservice"),
		atropos.WithServiceVersion("0.1.0"),
	)
	if err != nil {
		log.Warnf("failed to init atropos: %v", err)
	}
	if shutdown != nil {
		defer shutdown(ctx)
	}

	if profilerEnabled() {
		log.Info("Profiling enabled.")
		go initProfiling(log, "paymentservice", "1.0.0")
	} else {
		log.Info("Profiling disabled.")
	}

	port := defaultPort
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	svc := &service{log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /_healthz", svc.healthz)
	mux.HandleFunc("POST /charge", svc.charge)

	handler := atropos.IngressMiddleware(mux, "paymentservice")

	addr := ":" + port
	log.Infof("starting HTTP server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func newLogger() *logrus.Logger {
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
	return log
}

func profilerEnabled() bool {
	if os.Getenv("DISABLE_PROFILER") == "1" {
		return false
	}
	return os.Getenv("ENABLE_PROFILER") == "1"
}

func initProfiling(log logrus.FieldLogger, serviceName, version string) {
	for i := 1; i <= 3; i++ {
		log = log.WithField("retry", i)
		if err := profiler.Start(profiler.Config{
			Service:        serviceName,
			ServiceVersion: version,
		}); err != nil {
			log.Warnf("warn: failed to start profiler: %+v", err)
		} else {
			log.Info("started profiler")
			return
		}
		d := time.Second * 10 * time.Duration(i)
		log.Debugf("sleeping %v to retry initializing profiler", d)
		time.Sleep(d)
	}
	log.Warn("warning: could not initialize profiler after retrying, giving up")
}

func (s *service) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "SERVING"})
}

func (s *service) charge(w http.ResponseWriter, r *http.Request) {
	var req chargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	// Validate credit card
	cardType, err := validateCreditCard(req.CreditCard)
	if err != nil {
		s.log.Warnf("Credit card validation failed: %v", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log success
	s.log.WithFields(logrus.Fields{
		"amount":    fmt.Sprintf("%s%d.%d", req.Amount.CurrencyCode, req.Amount.Units, req.Amount.Nanos),
		"card_type": cardType,
		"last_four": lastFour(req.CreditCard.CreditCardNumber),
	}).Info("Transaction processed")

	txID := uuid.New().String()
	writeJSON(w, http.StatusOK, chargeResponse{TransactionID: txID})
}

func validateCreditCard(info creditCardInfo) (string, error) {
	// Check expiry
	now := time.Now()
	expYear := int(info.CreditCardExpirationYear)
	expMonth := int(info.CreditCardExpirationMonth)

	// If year is 2 digits, assume 20xx
	if expYear < 100 {
		expYear += 2000
	}

	expiry := time.Date(expYear, time.Month(expMonth)+1, 0, 0, 0, 0, 0, time.UTC)
	if now.After(expiry) {
		return "", fmt.Errorf("credit card expired")
	}

	// Check card number using regex
	number := strings.ReplaceAll(info.CreditCardNumber, "-", "")

	visaRegex := regexp.MustCompile(`^4[0-9]{12}(?:[0-9]{3})?$`)
	masterRegex := regexp.MustCompile(`^5[1-5][0-9]{14}$`)

	var cardType string
	if visaRegex.MatchString(number) {
		cardType = "visa"
	} else if masterRegex.MatchString(number) {
		cardType = "mastercard"
	} else {
		return "", fmt.Errorf("credit card not recognized: only visa or mastercard accepted")
	}

	if !luhnCheck(number) {
		return "", fmt.Errorf("invalid credit card number (luhn check failed)")
	}

	return cardType, nil
}

func luhnCheck(number string) bool {
	sum := 0
	alt := false
	for i := len(number) - 1; i >= 0; i-- {
		n, _ := strconv.Atoi(string(number[i]))
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return (sum % 10) == 0
}

func lastFour(s string) string {
	if len(s) < 4 {
		return s
	}
	return s[len(s)-4:]
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
