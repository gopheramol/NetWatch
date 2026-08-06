// Package speedtest measures Internet bandwidth on a schedule or on demand.
// The actual measurement is behind the Provider interface so the backing
// implementation (Ookla, iperf3, a custom server, ...) can be swapped later
// without touching callers.
package speedtest

import (
	"context"

	"github.com/gopheramol/NetWatch/internal/models"
)

// Provider runs a single bandwidth measurement and returns the result.
// Implementations own their own server selection and are expected to be
// safe to reuse across multiple calls.
type Provider interface {
	RunTest(ctx context.Context) (*models.SpeedTestResult, error)
}
