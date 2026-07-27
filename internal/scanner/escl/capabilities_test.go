package escl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRequestCapabilitiesRejectsOversizedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Write([]byte("<ScannerCapabilities>" + strings.Repeat(" ", maxCapabilitiesBodySize) + "</ScannerCapabilities>"))
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/eSCL")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := requestCapabilities(context.Background(), server.Client(), baseURL); err == nil {
		t.Fatal("requestCapabilities accepted an oversized response")
	}
}

func TestMapCapabilitiesKeepsSettingsPerSource(t *testing.T) {
	var document scannerCapabilitiesXML
	document.Platen.InputCaps = inputCapsXML{
		MaxWidth:  2550,
		MaxHeight: 3507,
		Profiles: []settingProfileXML{{
			ColorModes:  []string{"RGB24"},
			Resolutions: []resolutionXML{{X: 300, Y: 300}},
		}},
	}
	document.ADF.Simplex = &inputCapsXML{
		MaxWidth:  2550,
		MaxHeight: 4200,
		Profiles: []settingProfileXML{{
			ColorModes:  []string{"BlackAndWhite1"},
			Resolutions: []resolutionXML{{X: 200, Y: 200}},
		}},
	}

	capabilities := mapCapabilities(document)
	if len(capabilities.Sources) != 2 {
		t.Fatalf("source count = %d, want 2", len(capabilities.Sources))
	}

	platen, adf := capabilities.Sources[0], capabilities.Sources[1]
	if platen.ID != "Platen" || len(platen.ColorModes) != 1 || platen.ColorModes[0] != "RGB24" || len(platen.Resolutions) != 1 || platen.Resolutions[0] != 300 {
		t.Fatalf("platen capabilities = %#v", platen)
	}
	if adf.ID != "ADF Simplex" || !adf.Feeder || adf.Duplex || len(adf.ColorModes) != 1 || adf.ColorModes[0] != "BlackAndWhite1" || len(adf.Resolutions) != 1 || adf.Resolutions[0] != 200 {
		t.Fatalf("ADF capabilities = %#v", adf)
	}
}
