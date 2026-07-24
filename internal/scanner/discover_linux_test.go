package scanner

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestParseScanimageDevices(t *testing.T) {
	output := []byte("device `airscan:e0:OfficeJet` is a eSCL HP OfficeJet Pro 9010\ndevice `test:0` is a Noname virtual scanner\n")

	scanners := parseScanimageDevices(output)
	if len(scanners) != 2 {
		t.Fatalf("scanner count = %d, want 2", len(scanners))
	}

	for index, want := range []Info{
		{Device: "airscan:e0:OfficeJet", Description: "eSCL HP OfficeJet Pro 9010"},
		{Device: "test:0", Description: "Noname virtual scanner"},
	} {
		if got := scanners[index].Info(); got != want {
			t.Fatalf("scanner %d info = %#v, want %#v", index, got, want)
		}
	}
}

func TestParseScanimageDevicesIgnoresNonDeviceOutput(t *testing.T) {
	scanners := parseScanimageDevices([]byte("No scanners were identified.\ninvalid device line\n"))
	if len(scanners) != 0 {
		t.Fatalf("scanner count = %d, want 0", len(scanners))
	}
}

func TestDiscoverReturnsCommandDiagnostics(t *testing.T) {
	_, err := discover(context.Background(), func(context.Context) ([]byte, []byte, error) {
		return nil, []byte("Access to resource has been denied"), errors.New("exit status 1")
	})
	if err == nil {
		t.Fatal("discover succeeded, want an error")
	}
	if !strings.Contains(err.Error(), "run scanimage -L: exit status 1: Access to resource has been denied") {
		t.Fatalf("error = %q, want command diagnostics", err)
	}
}

func TestDiscoverHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	_, err := discover(ctx, func(context.Context) ([]byte, []byte, error) {
		called = true
		return nil, nil, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if called {
		t.Fatal("list command was called for a cancelled context")
	}
}
