package escl

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/slobbe/collate/internal/collate"
)

func TestCreateScanJobRejectsExternalAbsoluteLocation(t *testing.T) {
	externalRequests := 0
	external := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		externalRequests++
	}))
	defer external.Close()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/eSCL/ScannerStatus":
			io.WriteString(response, `<ScannerStatus><State>Idle</State></ScannerStatus>`)
		case "/eSCL/ScanJobs":
			response.Header().Set("Location", external.URL+"/eSCL/ScanJobs/42")
			response.WriteHeader(http.StatusCreated)
		default:
			http.Error(response, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/eSCL")
	if err != nil {
		t.Fatal(err)
	}
	_, err = performScan(context.Background(), server.Client(), baseURL, t.TempDir(), collate.ScanOptions{
		Source:     "Platen",
		Mode:       "RGB24",
		Paper:      collate.Paper{WidthMicrometres: 210_000, HeightMicrometres: 297_000},
		Resolution: 300,
	})
	if err == nil {
		t.Fatal("performScan accepted an external Location")
	}
	if externalRequests != 0 {
		t.Fatalf("external scanner received %d follow-up requests", externalRequests)
	}
}

func TestScanJobLifecycle(t *testing.T) {
	const document = "%PDF-test"
	var created, downloaded, deleted bool

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/eSCL/ScanJobs":
			created = true
			response.Header().Set("Location", "/eSCL/ScanJobs/42")
			response.WriteHeader(http.StatusCreated)
		case request.Method == http.MethodGet && request.URL.Path == "/eSCL/ScanJobs/42/NextDocument":
			downloaded = true
			response.Header().Set("Content-Type", "application/pdf")
			io.WriteString(response, document)
		case request.Method == http.MethodDelete && request.URL.Path == "/eSCL/ScanJobs/42":
			deleted = true
			response.WriteHeader(http.StatusNoContent)
		default:
			http.Error(response, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/eSCL")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	jobURL, err := createScanJob(ctx, server.Client(), baseURL, collate.ScanOptions{
		Source:     "Platen",
		Mode:       "RGB24",
		Paper:      collate.Paper{WidthMicrometres: 210_000, HeightMicrometres: 297_000},
		Resolution: 300,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := jobURL.String(), server.URL+"/eSCL/ScanJobs/42"; got != want {
		t.Fatalf("job URL = %q, want %q", got, want)
	}

	path, err := downloadNextDocument(ctx, server.Client(), jobURL, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != document {
		t.Fatalf("document = %q, want %q", contents, document)
	}
	if err := deleteScanJob(ctx, server.Client(), jobURL); err != nil {
		t.Fatal(err)
	}

	if !created || !downloaded || !deleted {
		t.Fatalf("job lifecycle: created=%t downloaded=%t deleted=%t", created, downloaded, deleted)
	}
}
