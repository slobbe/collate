package collate

import (
	"context"
	"testing"

	"github.com/slobbe/collate/internal/scanner"
	"github.com/slobbe/collate/internal/scanner/escl"
)

func TestDiscoverScannersMapsESCLDevicesToScannerInfo(t *testing.T) {
	infos, err := discoverScanners(context.Background(), func(context.Context) ([]escl.Device, error) {
		return []escl.Device{
			{Name: "Brother MFC", ID: "http://brother.local/eSCL"},
			{Name: "HP OfficeJet", ID: "https://hp.local/eSCL"},
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	want := []scanner.Info{
		{Name: "Brother MFC", ID: "http://brother.local/eSCL"},
		{Name: "HP OfficeJet", ID: "https://hp.local/eSCL"},
	}
	if len(infos) != len(want) {
		t.Fatalf("scanner count = %d, want %d", len(infos), len(want))
	}
	for index := range want {
		if infos[index] != want[index] {
			t.Fatalf("scanner %d = %#v, want %#v", index, infos[index], want[index])
		}
	}
}
