package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// statusRecorder captures the status code for the metrics middleware.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// withMetrics wraps a handler so every request is observed by route.
func withMetrics(reg *registry, route string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next(rec, r)
		reg.observe(route, rec.status, time.Since(start))
	}
}

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	versionFlag := flag.String("version", "", "service version (overrides VERSION env)")
	flag.Parse()

	if *versionFlag != "" {
		version = *versionFlag
	} else if v := os.Getenv("VERSION"); v != "" {
		version = v
	}

	reg := newRegistry()
	mux := http.NewServeMux()
	mux.HandleFunc("/", withMetrics(reg, "/", handleRoot))
	mux.HandleFunc("/healthz", withMetrics(reg, "/healthz", handleHealthz))
	mux.HandleFunc("/readyz", withMetrics(reg, "/readyz", handleReadyz))
	mux.HandleFunc("/work", withMetrics(reg, "/work", handleWork))
	mux.HandleFunc("/echo", withMetrics(reg, "/echo", handleEcho))
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		var sb strings.Builder
		reg.render(&sb)
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(sb.String()))
	})

	log.Printf("skypod %s listening on %s", version, *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}
