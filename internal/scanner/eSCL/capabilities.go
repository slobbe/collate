package escl

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/slobbe/collate/internal/scanner"
)

type scannerCapabilitiesXML struct {
	XMLName xml.Name `xml:"ScannerCapabilities"`

	Platen struct {
		InputCaps inputCapsXML `xml:"PlatenInputCaps"`
	} `xml:"Platen"`

	ADF struct {
		Simplex *inputCapsXML `xml:"AdfSimplexInputCaps"`
		Duplex  *inputCapsXML `xml:"AdfDuplexInputCaps"`

		FeederCapacity int      `xml:"FeederCapacity"`
		Options        []string `xml:"AdfOptions>AdfOption"`
	} `xml:"Adf"`
}

type inputCapsXML struct {
	MinWidth       int `xml:"MinWidth"`
	MaxWidth       int `xml:"MaxWidth"`
	MinHeight      int `xml:"MinHeight"`
	MaxHeight      int `xml:"MaxHeight"`
	MaxScanRegions int `xml:"MaxScanRegions"`

	Profiles []settingProfileXML `xml:"SettingProfiles>SettingProfile"`
	Intents  []string            `xml:"SupportedIntents>Intent"`

	MaxOpticalXResolution int `xml:"MaxOpticalXResolution"`
	MaxOpticalYResolution int `xml:"MaxOpticalYResolution"`

	RiskyLeftMargin   int `xml:"RiskyLeftMargin"`
	RiskyRightMargin  int `xml:"RiskyRightMargin"`
	RiskyTopMargin    int `xml:"RiskyTopMargin"`
	RiskyBottomMargin int `xml:"RiskyBottomMargin"`

	MaxPhysicalWidth  int `xml:"MaxPhysicalWidth"`
	MaxPhysicalHeight int `xml:"MaxPhysicalHeight"`
}

type settingProfileXML struct {
	ColorModes []string `xml:"ColorModes>ColorMode"`

	DocumentFormats    []string `xml:"DocumentFormats>DocumentFormat"`
	DocumentFormatsExt []string `xml:"DocumentFormats>DocumentFormatExt"`

	Resolutions []resolutionXML `xml:"SupportedResolutions>DiscreteResolutions>DiscreteResolution"`

	ColorSpaces      []string `xml:"ColorSpaces>ColorSpace"`
	CCDChannels      []string `xml:"CcdChannels>CcdChannel"`
	BinaryRenderings []string `xml:"BinaryRenderings>BinaryRendering"`
}

type resolutionXML struct {
	X int `xml:"XResolution"`
	Y int `xml:"YResolution"`
}

func RequestCapabilities(ctx context.Context, client *http.Client, baseURL string) (scanner.Capabilities, error) {
	if err := ctx.Err(); err != nil {
		return scanner.Capabilities{}, err
	}

	endpoint := baseURL + "/ScannerCapabilities"

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return scanner.Capabilities{}, err
	}

	response, err := client.Do(req)
	if err != nil {
		return scanner.Capabilities{}, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return scanner.Capabilities{}, fmt.Errorf("get scanner capabilities: %d", response.StatusCode)
	}

	document, err := parseCapabilities(response.Body)
	if err != nil {
		return scanner.Capabilities{}, fmt.Errorf("parse scanner capabilities: %w", err)
	}

	return mapCapabilities(document), nil
}

func parseCapabilities(body io.Reader) (scannerCapabilitiesXML, error) {
	var document scannerCapabilitiesXML

	if err := xml.NewDecoder(body).Decode(&document); err != nil {
		return scannerCapabilitiesXML{}, fmt.Errorf("decode scanner capabilities: %w", err)
	}

	if document.XMLName.Local != "ScannerCapabilities" {
		return scannerCapabilitiesXML{}, fmt.Errorf(
			"expected ScannerCapabilities, got %q",
			document.XMLName.Local,
		)
	}

	return document, nil
}

func mapCapabilities(document scannerCapabilitiesXML) scanner.Capabilities {
	var capabilities scanner.Capabilities

	appendSource := func(id, name string, feeder, duplex bool, input inputCapsXML) {
		if input.MaxWidth == 0 || input.MaxHeight == 0 {
			return
		}

		source := scanner.Source{
			ID:     id,
			Name:   name,
			Feeder: feeder,
			Duplex: duplex,
		}
		source.Dimensions.MinWidth = ToMicrometres(input.MinWidth)
		source.Dimensions.MinHeight = ToMicrometres(input.MinHeight)
		source.Dimensions.MaxWidth = ToMicrometres(input.MaxWidth)
		source.Dimensions.MaxHeight = ToMicrometres(input.MaxHeight)

		seenModes := map[string]bool{}
		seenResolutions := map[int]bool{}
		for _, profile := range input.Profiles {
			for _, mode := range profile.ColorModes {
				if mode != "" && !seenModes[mode] {
					seenModes[mode] = true
					source.ColorModes = append(source.ColorModes, mode)
				}
			}
			for _, resolution := range profile.Resolutions {
				if resolution.X > 0 && resolution.X == resolution.Y && !seenResolutions[resolution.X] {
					seenResolutions[resolution.X] = true
					source.Resolutions = append(source.Resolutions, resolution.X)
				}
			}
		}

		capabilities.Sources = append(capabilities.Sources, source)
	}

	appendSource("Platen", "Flatbed", false, false, document.Platen.InputCaps)
	if document.ADF.Simplex != nil {
		appendSource("ADF Simplex", "ADF (single-sided)", true, false, *document.ADF.Simplex)
	}
	if document.ADF.Duplex != nil {
		appendSource("ADF Duplex", "ADF (double-sided)", true, true, *document.ADF.Duplex)
	}

	return capabilities
}
