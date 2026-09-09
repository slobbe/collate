package escl

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/slobbe/collate/internal/collate"
)

// AirScanner adapts an eSCL device to the generic scanner interface.
type AirScanner struct {
	device     Device
	client     *http.Client
	workingDir string
}

// New creates a scanner for a discovered eSCL device.
func New(device Device) *AirScanner {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	allowSelfSigned := device.BaseURL != nil && strings.EqualFold(device.BaseURL.Scheme, "https") && isLocalHost(device.BaseURL.Hostname())
	transport.TLSClientConfig.InsecureSkipVerify = allowSelfSigned

	client := &http.Client{Transport: transport}
	if allowSelfSigned {
		scheme, host := device.BaseURL.Scheme, device.BaseURL.Host
		client.CheckRedirect = func(request *http.Request, _ []*http.Request) error {
			if strings.EqualFold(request.URL.Scheme, scheme) && strings.EqualFold(request.URL.Host, host) {
				return nil
			}
			return errors.New("refuse scanner redirect to a different origin")
		}
	}

	return &AirScanner{
		device: device,
		client: client,
	}
}

func isLocalHost(host string) bool {
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".local") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast())
}

func (s *AirScanner) Info() collate.ScannerInfo {
	return collate.ScannerInfo{
		ID:   s.device.ID,
		Name: s.device.Name,
	}
}

func (s *AirScanner) Capabilities(ctx context.Context) (collate.Capabilities, error) {
	return requestCapabilities(ctx, s.client, s.device.BaseURL)
}

func (s *AirScanner) Scan(ctx context.Context, options collate.ScanOptions) (collate.ScanResult, error) {
	return performScan(ctx, s.client, s.device.BaseURL, s.workingDir, options)
}

// Provider discovers eSCL scanners and adapts them to the Collate scanner port.
type Provider struct{}

// NewProvider creates an eSCL scanner provider.
func NewProvider() *Provider {
	return &Provider{}
}

// Discover returns the eSCL scanners advertised on the local network.
func (*Provider) Discover(ctx context.Context) ([]collate.Scanner, error) {
	devices, err := Discover(ctx)
	if err != nil {
		return nil, err
	}

	scanners := make([]collate.Scanner, len(devices))
	for index, device := range devices {
		scanners[index] = New(device)
	}
	return scanners, nil
}

var _ collate.Scanner = (*AirScanner)(nil)
var _ collate.ScannerProvider = (*Provider)(nil)
