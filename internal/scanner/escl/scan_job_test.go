package escl

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/slobbe/collate/internal/scanner"
)

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
	jobURL, err := createScanJob(ctx, server.Client(), baseURL, scanner.ScanOptions{
		Source:     "Platen",
		Mode:       "RGB24",
		Paper:      scanner.PaperA4,
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
