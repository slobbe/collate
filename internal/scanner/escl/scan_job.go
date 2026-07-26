package escl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/slobbe/collate/internal/scanner"
)

func createScanJob(ctx context.Context, client *http.Client, baseURL *url.URL, options scanner.ScanOptions) (*url.URL, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ticket, err := buildScanRequest(options)
	if err != nil {
		return nil, fmt.Errorf("build scan request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL.JoinPath("ScanJobs").String(), bytes.NewReader(ticket))
	if err != nil {
		return nil, fmt.Errorf("create scan request: %w", err)
	}
	request.Header.Set("Content-Type", "application/xml")
	request.Header.Set("Accept", "application/pdf")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send scan request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status %s", response.Status)
	}

	location := response.Header.Get("Location")
	if location == "" {
		return nil, fmt.Errorf("response has no Location header")
	}

	jobURL, err := request.URL.Parse(location)
	if err != nil {
		return nil, fmt.Errorf("parse scan job location %q: %w", location, err)
	}
	return jobURL, nil
}

func downloadNextDocument(ctx context.Context, client *http.Client, jobURL *url.URL, workingDir string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, jobURL.JoinPath("NextDocument").String(), nil)
	if err != nil {
		return "", fmt.Errorf("create document request: %w", err)
	}
	request.Header.Set("Accept", "application/pdf")

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("download scanned document: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download scanned document: %s", response.Status)
	}

	document, err := os.CreateTemp(workingDir, "collate-scan-*.pdf")
	if err != nil {
		return "", fmt.Errorf("create scanned document: %w", err)
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
		return "", fmt.Errorf("save scanned document: %w", err)
	}
	if err := document.Close(); err != nil {
		return "", fmt.Errorf("close scanned document: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	cleanup = false
	return documentPath, nil
}

func deleteScanJob(ctx context.Context, client *http.Client, jobURL *url.URL) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, jobURL.String(), nil)
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
