package checker

import (
	"context"
	"fmt"
	"net"

	"github.com/christiandente/micro-health-checker/internal/config"
)

type tcpChecker struct{ address string }

func newTCP(cfg *config.TCPConfig) Checker { return &tcpChecker{address: cfg.Address} }

func (c *tcpChecker) Check(ctx context.Context) error {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", c.address)
	if err != nil {
		return fmt.Errorf("tcp connection failed: %w", err)
	}
	return conn.Close()
}

func (*tcpChecker) Close() error { return nil }
