package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// handleRoot identifies the service. Behind the chart it answers with
// the pod hostname, which makes it easy to see replicas in a browser.
func handleRoot(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	writeJSON(w, http.StatusOK, map[string]any{
		"service":  "skypod",
		"version":  version,
		"hostname": hostname,
	})
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

// maxWorkMs caps the CPU burn so /work cannot be abused into a full DoS.
const maxWorkMs = 2000

// handleWork burns roughly ms milliseconds of CPU. The load script
// hammers it to push the pods over the HPA CPU target.
func handleWork(w http.ResponseWriter, r *http.Request) {
	ms := 50
	if raw := r.URL.Query().Get("ms"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "ms must be a non-negative integer",
			})
			return
		}
		if n > maxWorkMs {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("ms capped at %d", maxWorkMs),
			})
			return
		}
		ms = n
	}

	start := time.Now()
	deadline := start.Add(time.Duration(ms) * time.Millisecond)
	// Busy loop with real arithmetic so the compiler cannot fold it away.
	x := 1.0
	iterations := 0
	for time.Now().Before(deadline) {
		x = x*1.0000007 + 0.0000003
		iterations++
	}
	_ = x

	writeJSON(w, http.StatusOK, map[string]any{
		"requested_ms": ms,
		"elapsed_ms":   time.Since(start).Milliseconds(),
		"iterations":   iterations,
	})
}

// handleEcho reflects the request back. Useful to check what a gateway
// or ingress added on the way in.
func handleEcho(w http.ResponseWriter, r *http.Request) {
	headers := map[string][]string{}
	for k, v := range r.Header {
		headers[k] = v
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"method":  r.Method,
		"path":    r.URL.RequestURI(),
		"headers": headers,
		"remote":  r.RemoteAddr,
	})
}
