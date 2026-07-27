package escl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNewUsesDedicatedClientForLocalHTTPSDevice(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Write([]byte("<ScannerCapabilities></ScannerCapabilities>"))
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/eSCL")
	if err != nil {
		t.Fatal(err)
	}
	scanner := New(Device{BaseURL: baseURL})
	if scanner.client == http.DefaultClient {
		t.Fatal("New reused http.DefaultClient")
	}
	if scanner.client.Timeout != scannerHTTPTimeout {
		t.Fatalf("client timeout = %s, want %s", scanner.client.Timeout, scannerHTTPTimeout)
	}
	transport, ok := scanner.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client transport = %T, want *http.Transport", scanner.client.Transport)
	}
	if transport == http.DefaultTransport {
		t.Fatal("New reused http.DefaultTransport")
	}
	if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("local HTTPS client does not allow the device's self-signed certificate")
	}
	if _, err := scanner.Capabilities(context.Background()); err != nil {
		t.Fatalf("Capabilities over local self-signed HTTPS: %v", err)
	}

	publicURL, err := url.Parse("https://example.com/eSCL")
	if err != nil {
		t.Fatal(err)
	}
	publicTransport := New(Device{BaseURL: publicURL}).client.Transport.(*http.Transport)
	if publicTransport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("public HTTPS client disables certificate verification")
	}
}
