package escl

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slobbe/collate/internal/scanner"
)

const statusPollInterval = 500 * time.Millisecond

type ScanResult struct {
	path string
}

func (r *ScanResult) SavePDF(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r == nil || r.path == "" {
		return fmt.Errorf("no scanned document to save")
	}
	if path == "" {
		return fmt.Errorf("output path is required")
	}

	source, err := os.Open(r.path)
	if err != nil {
		return fmt.Errorf("open scanned document: %w", err)
	}
	defer source.Close()

	temporary, err := os.CreateTemp(filepath.Dir(path), ".collate-scan-*.pdf")
	if err != nil {
		return fmt.Errorf("create temporary output file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if _, err := io.Copy(temporary, source); err != nil {
		temporary.Close()
		return fmt.Errorf("copy scanned document: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary output file: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("save scanned document: %w", err)
	}

	return nil
}

func (r *ScanResult) Close() error {
	if r == nil || r.path == "" {
		return nil
	}
	return os.Remove(r.path)
}

var _ scanner.ScanResult = (*ScanResult)(nil)

func PerformScan(ctx context.Context, client *http.Client, baseURL, workingDir string, scanOptions *scanner.ScanOptions) (*ScanResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := waitForIdle(ctx, client, baseURL); err != nil {
		return nil, err
	}

	jobURL, err := createScanJob(ctx, client, baseURL, scanOptions)
	if err != nil {
		return nil, fmt.Errorf("create scan job: %w", err)
	}

	nextDocumentURL := strings.TrimRight(jobURL, "/") + "/NextDocument"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, nextDocumentURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create document request: %w", err)
	}
	request.Header.Set("Accept", "application/pdf")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download scanned document: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download scanned document: %s", response.Status)
	}

	document, err := os.CreateTemp(workingDir, "collate-scan-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("create scanned document: %w", err)
	}
	documentPath := document.Name()
	cleanup := true
	defer func() {
		if cleanup {
			os.Remove(documentPath)
		}
	}()

	if _, err := io.Copy(document, response.Body); err != nil {
		document.Close()
		return nil, fmt.Errorf("save scanned document: %w", err)
	}
	if err := document.Close(); err != nil {
		return nil, fmt.Errorf("close scanned document: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := deleteScanJob(ctx, client, jobURL); err != nil {
		return nil, err
	}

	cleanup = false
	return &ScanResult{path: documentPath}, nil
}

func deleteScanJob(ctx context.Context, client *http.Client, jobURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, jobURL, nil)
	if err != nil {
		return fmt.Errorf("create scan job cleanup request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("delete scan job: %w", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK, http.StatusNoContent, http.StatusNotFound:
		return nil
	default:
		return fmt.Errorf("delete scan job: unexpected status %s", response.Status)
	}
}

func waitForIdle(ctx context.Context, client *http.Client, baseURL string) error {
	for {
		state, err := GetStatus(ctx, client, baseURL)
		if err != nil {
			return fmt.Errorf("get scanner status: %w", err)
		}
		if state == "Idle" {
			return nil
		}

		timer := time.NewTimer(statusPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func createScanJob(ctx context.Context, client *http.Client, baseURL string, scanOptions *scanner.ScanOptions) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	scanRequest, err := buildScanRequest(scanOptions)
	if err != nil {
		return "", fmt.Errorf("build scan request: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/ScanJobs"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(scanRequest))
	if err != nil {
		return "", fmt.Errorf("create scan request: %w", err)
	}

	request.Header.Set("Content-Type", "application/xml")
	request.Header.Set("Accept", "application/pdf")

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("send scan request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create scan job: unexpected status %s", response.Status)
	}

	location := response.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("create scan job: response has no Location header")
	}

	locationURL, err := request.URL.Parse(location)
	if err != nil {
		return "", fmt.Errorf("parse scan job location %q: %w", location, err)
	}

	jobURL := locationURL.String()

	return jobURL, nil
}

func buildScanRequest(scanOptions *scanner.ScanOptions) ([]byte, error) {
	if scanOptions == nil {
		return nil, fmt.Errorf("scan options are required")
	}
	if scanOptions.Mode == "" {
		return nil, fmt.Errorf("scan mode is required")
	}
	if scanOptions.Resolution <= 0 {
		return nil, fmt.Errorf("scan resolution must be positive")
	}
	if scanOptions.Paper.WidthMicrometres <= 0 || scanOptions.Paper.HeightMicrometres <= 0 {
		return nil, fmt.Errorf("paper dimensions must be positive")
	}

	inputSource := ""
	duplex := ""
	switch scanOptions.Source {
	case "Platen":
		inputSource = "Platen"
	case "ADF Simplex":
		inputSource = "Feeder"
		duplex = "false"
	case "ADF Duplex":
		inputSource = "Feeder"
		duplex = "true"
	default:
		return nil, fmt.Errorf("unsupported scan source %q", scanOptions.Source)
	}

	var mode strings.Builder
	if err := xml.EscapeText(&mode, []byte(scanOptions.Mode)); err != nil {
		return nil, fmt.Errorf("escape scan mode: %w", err)
	}

	width := ToEsclUnits(scanOptions.Paper.WidthMicrometres)
	height := ToEsclUnits(scanOptions.Paper.HeightMicrometres)

	var request strings.Builder
	fmt.Fprintf(&request, `<?xml version="1.0" encoding="UTF-8"?>
<scan:ScanSettings xmlns:scan="http://schemas.hp.com/imaging/escl/2011/05/03" xmlns:escl="http://schemas.hp.com/imaging/escl/2011/05/03" xmlns:pwg="http://www.pwg.org/schemas/2010/12/sm">
  <pwg:Version>2.0</pwg:Version>
  <pwg:ScanRegions>
    <pwg:ScanRegion>
      <pwg:ContentRegionUnits>escl:ThreeHundredthsOfInches</pwg:ContentRegionUnits>
      <pwg:XOffset>0</pwg:XOffset>
      <pwg:YOffset>0</pwg:YOffset>
      <pwg:Width>%d</pwg:Width>
      <pwg:Height>%d</pwg:Height>
    </pwg:ScanRegion>
  </pwg:ScanRegions>
  <pwg:InputSource>%s</pwg:InputSource>
  <scan:ColorMode>%s</scan:ColorMode>
  <pwg:DocumentFormat>application/pdf</pwg:DocumentFormat>
  <scan:XResolution>%d</scan:XResolution>
  <scan:YResolution>%d</scan:YResolution>`, width, height, inputSource, mode.String(), scanOptions.Resolution, scanOptions.Resolution)
	if duplex != "" {
		fmt.Fprintf(&request, "\n  <scan:Duplex>%s</scan:Duplex>", duplex)
	}
	request.WriteString("\n</scan:ScanSettings>")

	return []byte(request.String()), nil
}
