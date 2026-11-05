package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
	mux.HandleFunc("/version", withMetrics(reg, "/version", handleVersion))
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		var sb strings.Builder
		reg.render(&sb)
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(sb.String()))
	})

	srv := &http.Server{Addr: *addr, Handler: mux}

	// On SIGTERM, Kubernetes is about to take this pod away. Fail the
	// readiness probe first so endpoints drain, give that a moment to
	// land, then let in-flight requests finish before exiting.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		ready.Store(false)
		log.Print("shutdown requested, draining")
		time.Sleep(2 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("skypod %s listening on %s", version, *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
