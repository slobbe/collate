package escl

import (
	"net"
	"net/url"
	"testing"

	"github.com/hashicorp/mdns"
)

func TestDeviceFromEntry(t *testing.T) {
	for _, test := range []struct {
		name     string
		entry    mdns.ServiceEntry
		scheme   string
		wantID   string
		wantName string
		ok       bool
	}{
		{
			name: "http default path",
			entry: mdns.ServiceEntry{
				Name:       "Brother._uscan._tcp.local.",
				Host:       "brother.local.",
				Port:       80,
				InfoFields: []string{"rs=eSCL", "ty=Brother MFC"},
			},
			scheme:   "http",
			wantID:   "http://brother.local/eSCL",
			wantName: "Brother MFC",
			ok:       true,
		},
		{
			name: "https custom path and port",
			entry: mdns.ServiceEntry{
				Name:       "Scanner._uscans._tcp.local.",
				Host:       "scanner.local.",
				Port:       8443,
				InfoFields: []string{"rs=/scan/escl/"},
			},
			scheme:   "https",
			wantID:   "https://scanner.local:8443/scan/escl",
			wantName: "Scanner._uscans._tcp.local",
			ok:       true,
		},
		{
			name: "default resource",
			entry: mdns.ServiceEntry{
				Host: "scanner.local.",
				Port: 80,
			},
			scheme: "http",
			wantID: "http://scanner.local/eSCL",
			ok:     true,
		},
		{
			name: "rejects absolute resource URL",
			entry: mdns.ServiceEntry{
				Host:       "scanner.local.",
				Port:       80,
				InfoFields: []string{"rs=https://example.com/eSCL"},
			},
			scheme: "http",
			ok:     false,
		},
		{
			name: "missing host",
			entry: mdns.ServiceEntry{
				Port: 80,
			},
			scheme: "http",
			ok:     false,
		},
		{
			name: "ipv6 default port",
			entry: mdns.ServiceEntry{
				Host: "fe80::1",
				Port: 80,
			},
			scheme: "http",
			wantID: "http://[fe80::1]/eSCL",
			ok:     true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := deviceFromEntry(&test.entry, test.scheme)
			if ok != test.ok {
				t.Fatalf("deviceFromEntry() ok = %t, want %t", ok, test.ok)
			}
			if !ok {
				return
			}
			if got.ID != test.wantID {
				t.Fatalf("device ID = %q, want %q", got.ID, test.wantID)
			}
			if got.Name != test.wantName {
				t.Fatalf("device name = %q, want %q", got.Name, test.wantName)
			}
		})
	}
}

func TestDeduplicateDevicesPrefersHTTPSForSameHostAndResource(t *testing.T) {
	httpURL, err := url.Parse("http://scanner.local:8080/eSCL")
	if err != nil {
		t.Fatal(err)
	}
	httpsURL, err := url.Parse("https://scanner.local:8443/eSCL")
	if err != nil {
		t.Fatal(err)
	}

	devices := deduplicateDevices([]Device{
		{ID: httpURL.String(), BaseURL: httpURL},
		{ID: httpsURL.String(), BaseURL: httpsURL},
	})
	if len(devices) != 1 {
		t.Fatalf("device count = %d, want 1", len(devices))
	}
	if devices[0].ID != httpsURL.String() {
		t.Fatalf("device ID = %q, want HTTPS endpoint %q", devices[0].ID, httpsURL)
	}
}

func TestDeviceFromEntryUsesAddressesOnlyForValidation(t *testing.T) {
	entry := mdns.ServiceEntry{
		Host:   "scanner.local.",
		Port:   80,
		AddrV4: net.ParseIP("192.0.2.1"),
	}

	device, ok := deviceFromEntry(&entry, "http")
	if !ok {
		t.Fatal("deviceFromEntry() failed")
	}
	if device.ID != "http://scanner.local/eSCL" {
		t.Fatalf("device ID = %q, want canonical hostname endpoint", device.ID)
	}
}
