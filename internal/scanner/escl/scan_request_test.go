package escl

import (
	"strings"
	"testing"

	"github.com/slobbe/collate/internal/collate"
)

func TestBuildScanRequestMapsSources(t *testing.T) {
	tests := []struct {
		name          string
		source        string
		inputSource   string
		duplexElement string
	}{
		{name: "platen", source: "Platen", inputSource: "Platen"},
		{name: "ADF simplex", source: "ADF Simplex", inputSource: "Feeder", duplexElement: "<scan:Duplex>false</scan:Duplex>"},
		{name: "ADF duplex", source: "ADF Duplex", inputSource: "Feeder", duplexElement: "<scan:Duplex>true</scan:Duplex>"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ticket, err := buildScanRequest(collate.ScanOptions{
				Source:     test.source,
				Mode:       "RGB24",
				Paper:      collate.Paper{WidthMicrometres: 210_000, HeightMicrometres: 297_000},
				Resolution: 300,
			})
			if err != nil {
				t.Fatal(err)
			}

			xml := string(ticket)
			if !strings.Contains(xml, "<pwg:InputSource>"+test.inputSource+"</pwg:InputSource>") {
				t.Fatalf("ticket does not select %q: %s", test.inputSource, xml)
			}
			if !strings.Contains(xml, "<pwg:Width>2480</pwg:Width>") || !strings.Contains(xml, "<pwg:Height>3508</pwg:Height>") {
				t.Fatalf("ticket does not contain A4 eSCL dimensions: %s", xml)
			}
			if test.duplexElement == "" {
				if strings.Contains(xml, "<scan:Duplex>") {
					t.Fatalf("platen ticket contains duplex setting: %s", xml)
				}
			} else if !strings.Contains(xml, test.duplexElement) {
				t.Fatalf("ticket does not contain %q: %s", test.duplexElement, xml)
			}
		})
	}
}

func TestBuildScanRequestRejectsInvalidOptions(t *testing.T) {
	paper := collate.Paper{WidthMicrometres: 210_000, HeightMicrometres: 297_000}
	tests := []collate.ScanOptions{
		{Source: "unknown", Mode: "RGB24", Paper: paper, Resolution: 300},
		{Source: "Platen", Paper: paper, Resolution: 300},
		{Source: "Platen", Mode: "RGB24", Paper: paper},
		{Source: "Platen", Mode: "RGB24", Resolution: 300},
	}

	for _, options := range tests {
		if _, err := buildScanRequest(options); err == nil {
			t.Fatalf("buildScanRequest(%#v) succeeded", options)
		}
	}
}
