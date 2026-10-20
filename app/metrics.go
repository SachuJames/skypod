// Package metrics is a tiny hand-rolled Prometheus registry: a request
// counter by route and status, and a duration summary per route. Stdlib
// only on purpose, nothing here needs a client library.
package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type counterKey struct {
	route  string
	status string
}

type routeStats struct {
	count        uint64
	statusCounts map[string]uint64
	totalNanos   uint64
}

type registry struct {
	mu    sync.Mutex
	stats map[string]*routeStats
}

func newRegistry() *registry {
	return &registry{stats: make(map[string]*routeStats)}
}

func (r *registry) observe(route string, status int, d time.Duration) {
	key := route
	if key == "" {
		key = "unknown"
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.stats[key]
	if !ok {
		s = &routeStats{statusCounts: make(map[string]uint64)}
		r.stats[key] = s
	}
	s.count++
	s.statusCounts[fmt.Sprint(status)]++
	s.totalNanos += uint64(d.Nanoseconds())
}

func (r *registry) render(sb *strings.Builder) {
	r.mu.Lock()
	routes := make([]string, 0, len(r.stats))
	for route := range r.stats {
		routes = append(routes, route)
	}
	r.mu.Unlock()
	sort.Strings(routes)

	sb.WriteString("# HELP skypod_http_requests_total Total HTTP requests handled.\n")
	sb.WriteString("# TYPE skypod_http_requests_total counter\n")
	for _, route := range routes {
		r.mu.Lock()
		s := r.stats[route]
		counts := make(map[string]uint64, len(s.statusCounts))
		for st, n := range s.statusCounts {
			counts[st] = n
		}
		r.mu.Unlock()
		statuses := make([]string, 0, len(counts))
		for st := range counts {
			statuses = append(statuses, st)
		}
		sort.Strings(statuses)
		for _, st := range statuses {
			fmt.Fprintf(sb, "skypod_http_requests_total{route=%q,status=%q} %d\n", route, st, counts[st])
		}
	}

	sb.WriteString("# HELP skypod_http_request_duration_seconds Request duration summary.\n")
	sb.WriteString("# TYPE skypod_http_request_duration_seconds summary\n")
	for _, route := range routes {
		r.mu.Lock()
		s := r.stats[route]
		count, total := s.count, s.totalNanos
		r.mu.Unlock()
		secs := float64(total) / 1e9
		fmt.Fprintf(sb, "skypod_http_request_duration_seconds_count{route=%q} %d\n", route, count)
		fmt.Fprintf(sb, "skypod_http_request_duration_seconds_sum{route=%q} %.6f\n", route, secs)
	}
}
