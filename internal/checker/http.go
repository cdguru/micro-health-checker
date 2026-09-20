package checker

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cdguru/micro-health-checker/internal/config"
)

const maxResponseBody = 1024 * 1024

type httpChecker struct {
	client   *http.Client
	method   string
	url      string
	headers  map[string]string
	statuses map[int]struct{}
	contains string
}

func newHTTP(cfg *config.HTTPConfig) Checker {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: cfg.InsecureSkipVerify} //nolint:gosec // explicitly configurable for private infrastructure
	client := &http.Client{Transport: transport}
	if cfg.FollowRedirects != nil && !*cfg.FollowRedirects {
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	}
	statuses := make(map[int]struct{}, len(cfg.ExpectedStatus))
	for _, status := range cfg.ExpectedStatus {
		statuses[status] = struct{}{}
	}
	return &httpChecker{
		client:   client,
		method:   cfg.Method,
		url:      cfg.URL,
		headers:  cfg.Headers,
		statuses: statuses,
		contains: cfg.BodyContains,
	}
}

func (c *httpChecker) Check(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, c.method, c.url, nil)
	if err != nil {
		return fmt.Errorf("build HTTP request: %w", err)
	}
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if len(c.statuses) == 0 {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
		}
	} else if _, ok := c.statuses[resp.StatusCode]; !ok {
		return fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	if c.contains == "" {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBody))
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return fmt.Errorf("read HTTP response: %w", err)
	}
	if !strings.Contains(string(body), c.contains) {
		return fmt.Errorf("HTTP response does not contain required text")
	}
	return nil
}

func (c *httpChecker) Close() error {
	c.client.CloseIdleConnections()
	return nil
}
