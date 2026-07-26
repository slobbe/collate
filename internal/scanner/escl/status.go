package escl

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
)

type scannerState string

const scannerStateIdle scannerState = "Idle"

type scannerStatusXML struct {
	State scannerState `xml:"State"`
}

func getStatus(ctx context.Context, client *http.Client, baseURL *url.URL) (scannerState, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL.JoinPath("ScannerStatus").String(), nil)
	if err != nil {
		return "", fmt.Errorf("create scanner status request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s", response.Status)
	}

	var status scannerStatusXML
	if err := xml.NewDecoder(response.Body).Decode(&status); err != nil {
		return "", fmt.Errorf("decode scanner status: %w", err)
	}
	return status.State, nil
}
