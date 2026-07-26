package escl

import (
	"context"
	"net/http"

	"github.com/slobbe/collate/internal/scanner"
)

// AirScanner adapts an eSCL device to the generic scanner interface.
type AirScanner struct {
	device     Device
	client     *http.Client
	workingDir string
}

// New creates a scanner for a discovered eSCL device.
func New(device Device) *AirScanner {
	return &AirScanner{
		device: device,
		client: http.DefaultClient,
	}
}

func (s *AirScanner) Info() scanner.Info {
	return scanner.Info{
		ID:   s.device.ID,
		Name: s.device.Name,
	}
}

func (s *AirScanner) Capabilities(ctx context.Context) (scanner.Capabilities, error) {
	return requestCapabilities(ctx, s.client, s.device.BaseURL)
}

func (s *AirScanner) Scan(ctx context.Context, options scanner.ScanOptions) (scanner.ScanResult, error) {
	return performScan(ctx, s.client, s.device.BaseURL, s.workingDir, options)
}

var _ scanner.Scanner = (*AirScanner)(nil)
