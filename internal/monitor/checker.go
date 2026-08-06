// Package monitor performs periodic Internet connectivity checks (DNS, HTTP,
// TCP reachability) and turns consecutive failures into detected outages,
// wiring the result into analytics and Telegram notifications.
package monitor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gopheramol/NetWatch/internal/config"
	"github.com/gopheramol/NetWatch/internal/models"
)

// Checker performs the three-pronged connectivity probe described in the
// project spec: DNS resolution, an HTTPS request, and a TCP reachability
// check used as a ping substitute (ICMP normally requires elevated
// privileges that a lightweight background service should not need).
type Checker struct {
	cfg        config.MonitorConfig
	httpClient *http.Client
	resolver   *net.Resolver
}

// NewChecker builds a Checker bound to the given monitor configuration.
func NewChecker(cfg config.MonitorConfig) *Checker {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	return &Checker{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: timeout},
		resolver:   net.DefaultResolver,
	}
}

// Check runs all three probes and returns a single connectivity result.
func (c *Checker) Check(ctx context.Context) models.ConnectivityCheck {
	timeout := time.Duration(c.cfg.TimeoutSeconds) * time.Second
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	now := time.Now()
	dnsOk, dnsErr := c.checkDNS(checkCtx)
	httpOk, httpLatency, httpErr := c.checkHTTP(checkCtx)
	tcpOk, tcpLatency, tcpErr := c.checkTCP(checkCtx)

	latency := httpLatency
	if !httpOk && tcpOk {
		latency = tcpLatency
	}

	successCount := boolToInt(dnsOk) + boolToInt(httpOk) + boolToInt(tcpOk)

	var status models.ConnectivityStatus
	var reasons []string
	switch successCount {
	case 3:
		status = models.StatusUp
	case 0:
		status = models.StatusDown
	default:
		status = models.StatusDegraded
	}

	if dnsErr != nil {
		reasons = append(reasons, fmt.Sprintf("dns: %v", dnsErr))
	}
	if httpErr != nil {
		reasons = append(reasons, fmt.Sprintf("http: %v", httpErr))
	}
	if tcpErr != nil {
		reasons = append(reasons, fmt.Sprintf("tcp: %v", tcpErr))
	}

	return models.ConnectivityCheck{
		Timestamp:     now,
		Status:        status,
		LatencyMs:     latency,
		DNSOk:         dnsOk,
		HTTPOk:        httpOk,
		PingOk:        tcpOk,
		FailureReason: strings.Join(reasons, "; "),
	}
}

func (c *Checker) checkDNS(ctx context.Context) (bool, error) {
	_, err := c.resolver.LookupHost(ctx, c.cfg.DNSHost)
	return err == nil, err
}

func (c *Checker) checkHTTP(ctx context.Context) (bool, float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.HTTPCheckURL, nil)
	if err != nil {
		return false, 0, fmt.Errorf("building request: %w", err)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	latency := float64(time.Since(start).Microseconds()) / 1000.0
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return false, latency, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return true, latency, nil
}

func (c *Checker) checkTCP(ctx context.Context) (bool, float64, error) {
	dialer := net.Dialer{}
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(c.cfg.PingHost, "53"))
	latency := float64(time.Since(start).Microseconds()) / 1000.0
	if err != nil {
		return false, 0, err
	}
	_ = conn.Close()
	return true, latency, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
