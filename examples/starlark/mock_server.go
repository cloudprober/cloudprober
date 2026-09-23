// A tiny mock token-auth trading-style API for testing the SCRIPT probe end-to-end.
//
//	go run mock_server.go              # listens on :8080
//	go run mock_server.go -addr :9000  # custom address
//
// Endpoints:
//
//	POST /api-token-auth/                  -> {"token": "tok-abc"}
//	GET  /accounts/    (Token tok-abc)     -> {"results": [{...}]}
//	GET  /portfolios/<acct>/               -> {"equity": "...", ...}
//	GET  /quotes/?symbols=XYZ,ABC          -> {"results": [{...}]}
//
// Pass -fail-rate=0.1 to randomly 503 ~10% of /accounts/ requests.

//go:build ignore

package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

var (
	addr     = flag.String("addr", ":8080", "address to listen on")
	failRate = flag.Float64("fail-rate", 0.0, "fraction of /accounts/ requests that 503")
)

const tokenValue = "tok-abc"

func main() {
	flag.Parse()

	mux := http.NewServeMux()

	mux.HandleFunc("/api-token-auth/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if body.Username == "" || body.Password == "" {
			http.Error(w, "missing creds", http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"token": tokenValue})
	})

	mux.HandleFunc("/accounts/", func(w http.ResponseWriter, r *http.Request) {
		if !checkToken(w, r) {
			return
		}
		if rand.Float64() < *failRate {
			http.Error(w, "transient", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"results": []map[string]any{{
				"account_number": "5QR12345",
				"buying_power":   "1234.56",
				"type":           "cash",
			}},
		})
	})

	mux.HandleFunc("/portfolios/", func(w http.ResponseWriter, r *http.Request) {
		if !checkToken(w, r) {
			return
		}
		acct := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/portfolios/"), "/")
		if acct == "" {
			http.Error(w, "missing account", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"account":      acct,
			"equity":       "5421.99",
			"market_value": "4998.50",
		})
	})

	mux.HandleFunc("/quotes/", func(w http.ResponseWriter, r *http.Request) {
		if !checkToken(w, r) {
			return
		}
		symbols := r.URL.Query().Get("symbols")
		if symbols == "" {
			http.Error(w, "missing symbols", http.StatusBadRequest)
			return
		}
		results := []map[string]any{}
		for _, s := range strings.Split(symbols, ",") {
			results = append(results, map[string]any{
				"symbol":           strings.TrimSpace(s),
				"last_trade_price": "180.50",
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"results": results})
	})

	log.Printf("mock trading-api listening on %s (fail-rate=%.2f)", *addr, *failRate)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func checkToken(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("Authorization") != "Token "+tokenValue {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
