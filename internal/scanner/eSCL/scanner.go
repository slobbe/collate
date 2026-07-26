package escl

import (
	"context"
	"net/http"

	"github.com/slobbe/collate/internal/scanner"
)

type Airscanner struct {
	name    string
	baseURL string
	client  *http.Client
}

func NewAirscanner(name string, baseURL string) *Airscanner {
	client := &http.Client{}

	return &Airscanner{
		name:    name,
		baseURL: baseURL,
		client:  client,
	}
}

func (s *Airscanner) Info() scanner.Info {
	return scanner.Info{
		ID:   s.baseURL,
		Name: s.name,
	}
}

func (s *Airscanner) Capabilities(ctx context.Context) (scanner.Capabilities, error) {
	capabilities, err := RequestCapabilities(ctx, s.client, s.baseURL)
	return capabilities, err
}

func (s *Airscanner) Scan(ctx context.Context, options scanner.ScanOptions) (scanner.ScanResult, error) {
	result, err := PerformScan(ctx, s.client, s.baseURL, "", &options)
	if err != nil {
		return nil, err
	}
	return result, nil
}

var _ scanner.Scanner = (*Airscanner)(nil)
