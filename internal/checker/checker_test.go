package checker

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/christiandente/micro-health-checker/internal/config"
)

func TestTCPChecker(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
		}
	}()
	instance := newTCP(&config.TCPConfig{Address: listener.Addr().String()})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := instance.Check(ctx); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestHTTPCheckerAssertions(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}))
	defer target.Close()
	instance := newHTTP(&config.HTTPConfig{
		URL: target.URL, Method: http.MethodGet, ExpectedStatus: []int{http.StatusAccepted}, BodyContains: "ready",
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := instance.Check(ctx); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestHTTPCheckerRejectsUnexpectedStatus(t *testing.T) {
	target := httptest.NewServer(http.NotFoundHandler())
	defer target.Close()
	instance := newHTTP(&config.HTTPConfig{URL: target.URL, Method: http.MethodGet, ExpectedStatus: []int{200}})
	if err := instance.Check(context.Background()); err == nil {
		t.Fatal("Check() expected status error")
	}
}
