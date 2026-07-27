package collate

import (
	"context"
	"fmt"
)

// ScanService coordinates scanner discovery and scan sessions.
type ScanService struct {
	provider ScannerProvider
}

// NewScanService creates a scanning application service.
func NewScanService(provider ScannerProvider) *ScanService {
	return &ScanService{provider: provider}
}

// DiscoverScanners returns scanners available through the configured provider.
func (s *ScanService) DiscoverScanners(ctx context.Context) ([]ScannerInfo, error) {
	scanners, err := s.provider.Discover(ctx)
	if err != nil {
		return nil, err
	}

	infos := make([]ScannerInfo, len(scanners))
	for index, scanner := range scanners {
		infos[index] = scanner.Info()
	}
	return infos, nil
}

// ScannerCapabilities returns the capabilities of a discovered scanner.
func (s *ScanService) ScannerCapabilities(ctx context.Context, deviceID string) (Capabilities, error) {
	scanner, err := s.scannerByID(ctx, deviceID)
	if err != nil {
		return Capabilities{}, err
	}
	return scanner.Capabilities(ctx)
}

// StartScanByID starts a scan using a discovered scanner.
func (s *ScanService) StartScanByID(ctx context.Context, deviceID string, options ScanOptions) (*ScanSession, error) {
	scanner, err := s.scannerByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	return StartScan(ctx, scanner, options)
}

func (s *ScanService) scannerByID(ctx context.Context, deviceID string) (Scanner, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.provider == nil {
		return nil, fmt.Errorf("scanner provider is required")
	}

	scanners, err := s.provider.Discover(ctx)
	if err != nil {
		return nil, err
	}
	for _, scanner := range scanners {
		if scanner.Info().ID == deviceID {
			return scanner, nil
		}
	}
	return nil, fmt.Errorf("scanner %q not found; run \"collate scan --device-list\" to list available scanners", deviceID)
}
