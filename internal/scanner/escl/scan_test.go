package escl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/slobbe/collate/internal/collate"
)

func TestPerformScanDeletesJobWhenNextDocumentFails(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/eSCL/ScannerStatus":
			response.Write([]byte("<ScannerStatus><State>Idle</State></ScannerStatus>"))
		case request.Method == http.MethodPost && request.URL.Path == "/eSCL/ScanJobs":
			response.Header().Set("Location", "/eSCL/ScanJobs/42")
			response.WriteHeader(http.StatusCreated)
		case request.Method == http.MethodGet && request.URL.Path == "/eSCL/ScanJobs/42/NextDocument":
			http.Error(response, "scan failed", http.StatusInternalServerError)
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
	_, err = performScan(context.Background(), server.Client(), baseURL, t.TempDir(), collate.ScanOptions{
		Source:     "Platen",
		Mode:       "RGB24",
		Paper:      collate.Paper{WidthMicrometres: 210_000, HeightMicrometres: 297_000},
		Resolution: 300,
	})
	if err == nil {
		t.Fatal("performScan succeeded when NextDocument failed")
	}
	if !deleted {
		t.Fatal("performScan did not delete the failed scan job")
	}
}
