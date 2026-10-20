package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	reg := newRegistry()
	mux := http.NewServeMux()
	mux.HandleFunc("/", withMetrics(reg, "/", handleRoot))
	mux.HandleFunc("/healthz", withMetrics(reg, "/healthz", handleHealthz))
	mux.HandleFunc("/readyz", withMetrics(reg, "/readyz", handleReadyz))
	mux.HandleFunc("/work", withMetrics(reg, "/work", handleWork))
	mux.HandleFunc("/echo", withMetrics(reg, "/echo", handleEcho))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func getBody(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, string(body)
}

func TestRoot(t *testing.T) {
	srv := newTestServer(t)
	code, body := getBody(t, srv.URL+"/")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	for _, want := range []string{`"service":"skypod"`, `"version"`, `"hostname"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %s: %s", want, body)
		}
	}
}

func TestProbes(t *testing.T) {
	srv := newTestServer(t)
	for _, path := range []string{"/healthz", "/readyz"} {
		code, body := getBody(t, srv.URL+path)
		if code != http.StatusOK || !strings.Contains(body, "ok") {
			t.Errorf("%s: got %d %q", path, code, body)
		}
	}
}

func TestEcho(t *testing.T) {
	srv := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/echo?x=1", nil)
	req.Header.Set("X-Test-Marker", "hello")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	for _, want := range []string{`"method":"GET"`, "X-Test-Marker", `"path":"/echo?x=1"`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("echo body missing %s: %s", want, body)
		}
	}
}

func TestMetricsCounters(t *testing.T) {
	reg := newRegistry()
	mux := http.NewServeMux()
	mux.HandleFunc("/work", withMetrics(reg, "/work", handleWork))
	mux.HandleFunc("/missing", withMetrics(reg, "/missing", http.NotFound))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	if _, err := http.Get(srv.URL + "/work?ms=5"); err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	reg.render(&sb)
	out := sb.String()
	for _, want := range []string{
		"skypod_http_requests_total",
		`route="/work"`,
		`status="200"`,
		"skypod_http_request_duration_seconds_count",
		`route="/work"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("metrics output missing %q:\n%s", want, out)
		}
	}
}

func TestWorkBounds(t *testing.T) {
	srv := newTestServer(t)
	code, body := getBody(t, srv.URL+"/work?ms=5")
	if code != http.StatusOK {
		t.Fatalf("ms=5: status %d body %s", code, body)
	}
	for _, q := range []string{"/work?ms=99999", "/work?ms=abc", "/work?ms=-5"} {
		code, _ := getBody(t, srv.URL+q)
		if code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", q, code)
		}
	}
}

func TestWorkBurnsAtLeastRequested(t *testing.T) {
	srv := newTestServer(t)
	code, body := getBody(t, srv.URL+"/work?ms=30")
	if code != http.StatusOK {
		t.Fatalf("status %d body %s", code, body)
	}
	if !strings.Contains(body, `"requested_ms":30`) {
		t.Errorf("body missing requested_ms: %s", body)
	}
	if !strings.Contains(body, `"iterations"`) {
		t.Errorf("body missing iterations: %s", body)
	}
}
