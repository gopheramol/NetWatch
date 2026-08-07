package speedtest

import (
	"context"
	"fmt"

	"github.com/gopheramol/NetWatch/internal/models"
	speedtestgo "github.com/showwin/speedtest-go/speedtest"
)

// OoklaProvider implements Provider on top of the public speedtest.net
// network via the showwin/speedtest-go client library.
type OoklaProvider struct {
	client *speedtestgo.Speedtest
}

// NewOoklaProvider builds the default speedtest.net-backed Provider.
func NewOoklaProvider() *OoklaProvider {
	return &OoklaProvider{client: speedtestgo.New()}
}

// RunTest selects the best reachable server and measures ping, jitter,
// download, and upload throughput against it, automatically retrying
// candidate servers if a server returns zero speed or fails to respond.
func (p *OoklaProvider) RunTest(ctx context.Context) (*models.SpeedTestResult, error) {
	user, err := p.client.FetchUserInfoContext(ctx)
	if err != nil {
		user = &speedtestgo.User{}
	}

	serverList, err := p.client.FetchServerListContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching speedtest servers: %w", err)
	}

	targets, err := serverList.FindServer(nil)
	if err != nil || len(targets) == 0 {
		return nil, fmt.Errorf("no reachable speedtest server found: %w", err)
	}

	candidateServers := targets
	if avail := targets.Available(); avail != nil && len(*avail) > 0 {
		candidateServers = *avail
	}

	var lastErr error
	maxAttempts := 3
	if len(candidateServers) < maxAttempts {
		maxAttempts = len(candidateServers)
	}

	for i := 0; i < maxAttempts; i++ {
		server := candidateServers[i]

		if err := server.PingTestContext(ctx, nil); err != nil {
			lastErr = fmt.Errorf("server %s ping test: %w", server.Host, err)
			continue
		}

		if err := server.DownloadTestContext(ctx); err != nil {
			lastErr = fmt.Errorf("server %s download test: %w", server.Host, err)
			continue
		}

		if err := server.UploadTestContext(ctx); err != nil {
			lastErr = fmt.Errorf("server %s upload test: %w", server.Host, err)
			continue
		}

		dlMbps := server.DLSpeed.Mbps()
		ulMbps := server.ULSpeed.Mbps()

		// Retry with next server if zero or invalid throughput was reported
		if dlMbps <= 0 && ulMbps <= 0 {
			lastErr = fmt.Errorf("server %s returned 0 Mbps throughput", server.Host)
			continue
		}

		return &models.SpeedTestResult{
			DownloadMbps: dlMbps,
			UploadMbps:   ulMbps,
			PingMs:       float64(server.Latency.Microseconds()) / 1000.0,
			JitterMs:     float64(server.Jitter.Microseconds()) / 1000.0,
			ISP:          user.Isp,
			Server:       server.Name + " (" + server.Sponsor + ")",
			Provider:     "ookla",
		}, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("speedtest failed after testing candidate servers: %w", lastErr)
	}
	return nil, fmt.Errorf("no valid speedtest result obtained")
}
