package escl

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/slobbe/collate/internal/collate"
)

const (
	statusPollInterval    = 500 * time.Millisecond
	scanJobCleanupTimeout = 5 * time.Second
)

func performScan(
	ctx context.Context,
	client *http.Client,
	baseURL *url.URL,
	workingDir string,
	options collate.ScanOptions,
) (*scanResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if baseURL == nil {
		return nil, fmt.Errorf("scanner URL is required")
	}
	if err := waitForIdle(ctx, client, baseURL); err != nil {
		return nil, err
	}

	jobURL, err := createScanJob(ctx, client, baseURL, options)
	if err != nil {
		return nil, fmt.Errorf("create scan job: %w", err)
	}
	jobDeleted := false
	defer func() {
		if jobDeleted {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), scanJobCleanupTimeout)
		defer cancel()
		_ = deleteScanJob(cleanupCtx, client, jobURL)
	}()

	documentPath, err := downloadNextDocument(ctx, client, jobURL, workingDir)
	if err != nil {
		return nil, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			os.Remove(documentPath)
		}
	}()

	if err := deleteScanJob(ctx, client, jobURL); err != nil {
		return nil, err
	}
	jobDeleted = true

	cleanup = false
	return &scanResult{path: documentPath}, nil
}

func waitForIdle(ctx context.Context, client *http.Client, baseURL *url.URL) error {
	for {
		state, err := getStatus(ctx, client, baseURL)
		if err != nil {
			return fmt.Errorf("get scanner status: %w", err)
		}
		if state == scannerStateIdle {
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
