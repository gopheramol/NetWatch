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

// RunTest selects the nearest reachable server and measures ping, jitter,
// download, and upload throughput against it.
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
	server := targets[0]
	defer server.Context.Reset()

	if err := server.PingTestContext(ctx, nil); err != nil {
		return nil, fmt.Errorf("ping test: %w", err)
	}
	if err := server.DownloadTestContext(ctx); err != nil {
		return nil, fmt.Errorf("download test: %w", err)
	}
	if err := server.UploadTestContext(ctx); err != nil {
		return nil, fmt.Errorf("upload test: %w", err)
	}

	return &models.SpeedTestResult{
		DownloadMbps: server.DLSpeed.Mbps(),
		UploadMbps:   server.ULSpeed.Mbps(),
		PingMs:       float64(server.Latency.Microseconds()) / 1000.0,
		JitterMs:     float64(server.Jitter.Microseconds()) / 1000.0,
		ISP:          user.Isp,
		Server:       server.Name + " (" + server.Sponsor + ")",
		Provider:     "ookla",
	}, nil
}
