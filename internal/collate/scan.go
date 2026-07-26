package collate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/slobbe/collate/internal/scanner"
	escl "github.com/slobbe/collate/internal/scanner/eSCL"
)

type discoverESCLDevices func(context.Context) ([]escl.Device, error)

// DiscoverScanners returns scanners available to the active scanning backend.
func DiscoverScanners(ctx context.Context) ([]scanner.Info, error) {
	return discoverScanners(ctx, escl.Discover)
}

func discoverScanners(ctx context.Context, discover discoverESCLDevices) ([]scanner.Info, error) {
	devices, err := discover(ctx)
	if err != nil {
		return nil, err
	}

	infos := make([]scanner.Info, len(devices))
	for index, device := range devices {
		infos[index] = scanner.Info{
			ID:   device.ID,
			Name: device.Name,
		}
	}
	return infos, nil
}

// ScannerCapabilities returns the capabilities of a discovered scanner.
func ScannerCapabilities(ctx context.Context, deviceID string) (scanner.Capabilities, error) {
	return scannerCapabilities(ctx, deviceID, escl.Discover)
}

func scannerCapabilities(ctx context.Context, deviceID string, discover discoverESCLDevices) (scanner.Capabilities, error) {
	deviceScanner, err := scannerByID(ctx, deviceID, discover)
	if err != nil {
		return scanner.Capabilities{}, err
	}
	return deviceScanner.Capabilities(ctx)
}

// StartScanByID starts a scan using the active scanning backend.
func StartScanByID(ctx context.Context, deviceID string, options scanner.ScanOptions) (*ScanSession, error) {
	return startScanByID(ctx, deviceID, options, escl.Discover)
}

func startScanByID(ctx context.Context, deviceID string, options scanner.ScanOptions, discover discoverESCLDevices) (*ScanSession, error) {
	deviceScanner, err := scannerByID(ctx, deviceID, discover)
	if err != nil {
		return nil, err
	}
	return StartScan(ctx, deviceScanner, options)
}

func scannerByID(ctx context.Context, deviceID string, discover discoverESCLDevices) (scanner.Scanner, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	devices, err := discover(ctx)
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		if device.ID != deviceID {
			continue
		}
		return escl.NewAirscanner(device.Name, device.BaseURL.String()), nil
	}
	return nil, fmt.Errorf("scanner %q not found; run \"collate scan --device-list\" to list available scanners", deviceID)
}

// ScanSession coordinates front and optional back scans for one document.
type ScanSession struct {
	scanner scanner.Scanner
	options scanner.ScanOptions
	front   scanner.ScanResult
	back    scanner.ScanResult
}

// StartScan acquires the front pages of a document.
func StartScan(ctx context.Context, device scanner.Scanner, options scanner.ScanOptions) (*ScanSession, error) {
	if device == nil {
		return nil, fmt.Errorf("scanner is required")
	}

	front, err := device.Scan(ctx, options)
	if err != nil {
		return nil, err
	}

	return &ScanSession{
		scanner: device,
		options: options,
		front:   front,
	}, nil
}

// ScanBack acquires the back pages of the document.
func (s *ScanSession) ScanBack(ctx context.Context) error {
	if s.back != nil {
		return fmt.Errorf("back pages have already been scanned")
	}

	back, err := s.scanner.Scan(ctx, s.options)
	if err != nil {
		return err
	}
	s.back = back
	return nil
}

// SaveFront saves the front pages without duplex collation.
func (s *ScanSession) SaveFront(ctx context.Context, outputPath string) error {
	if s.front == nil {
		return fmt.Errorf("front pages have not been scanned")
	}
	return s.front.SavePDF(ctx, outputPath)
}

// Collate saves the front and back pages temporarily, then creates a duplex PDF.
func (s *ScanSession) Collate(ctx context.Context, outputPath string, backOrder BackOrder) error {
	if s.front == nil {
		return fmt.Errorf("front pages have not been scanned")
	}
	if s.back == nil {
		return fmt.Errorf("back pages have not been scanned")
	}

	temporaryDir, err := os.MkdirTemp(filepath.Dir(outputPath), ".collate-duplex-")
	if err != nil {
		return fmt.Errorf("create temporary scan directory: %w", err)
	}
	defer os.RemoveAll(temporaryDir)

	frontPath := filepath.Join(temporaryDir, "front.pdf")
	if err := s.front.SavePDF(ctx, frontPath); err != nil {
		return fmt.Errorf("save front pages: %w", err)
	}

	backPath := filepath.Join(temporaryDir, "back.pdf")
	if err := s.back.SavePDF(ctx, backPath); err != nil {
		return fmt.Errorf("save back pages: %w", err)
	}

	return Collate(ctx, frontPath, backPath, outputPath, backOrder)
}

// Close releases resources held by the front and back scan results.
func (s *ScanSession) Close() error {
	var errs []error
	if s.front != nil {
		errs = append(errs, s.front.Close())
	}
	if s.back != nil {
		errs = append(errs, s.back.Close())
	}
	return errors.Join(errs...)
}
