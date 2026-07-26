package escl

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
)

type ScannerStatus struct {
	State string `xml:"State"`
}

func GetStatus(ctx context.Context, client *http.Client, baseURL string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	endpoint := baseURL + "/ScannerStatus"

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}

	response, err := client.Do(req)

	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get scanner status: unexpected status %s", response.Status)
	}

	var status ScannerStatus
	if err := xml.NewDecoder(response.Body).Decode(&status); err != nil {
		return "", fmt.Errorf("decode scanner status: %w", err)
	}

	return status.State, nil
}
