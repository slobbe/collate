package collate

import (
	"context"
	"testing"
)

type scannerProviderStub struct {
	scanners []Scanner
	err      error
}

func (p scannerProviderStub) Discover(context.Context) ([]Scanner, error) {
	return p.scanners, p.err
}

type scannerStub struct {
	info ScannerInfo
}

func (s scannerStub) Info() ScannerInfo {
	return s.info
}

func (scannerStub) Capabilities(context.Context) (Capabilities, error) {
	return Capabilities{}, nil
}

func (scannerStub) Scan(context.Context, ScanOptions) (ScanResult, error) {
	return nil, nil
}

func TestScanServiceDiscoversScannerInfo(t *testing.T) {
	service := NewScanService(scannerProviderStub{scanners: []Scanner{
		scannerStub{info: ScannerInfo{Name: "Brother MFC", ID: "http://brother.local/eSCL"}},
		scannerStub{info: ScannerInfo{Name: "HP OfficeJet", ID: "https://hp.local/eSCL"}},
	}})

	infos, err := service.DiscoverScanners(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	want := []ScannerInfo{
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
