package escl

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/slobbe/collate/internal/scanner"
)

func buildScanRequest(options scanner.ScanOptions) ([]byte, error) {
	if options.Mode == "" {
		return nil, fmt.Errorf("scan mode is required")
	}
	if options.Resolution <= 0 {
		return nil, fmt.Errorf("scan resolution must be positive")
	}
	if options.Paper.WidthMicrometres <= 0 || options.Paper.HeightMicrometres <= 0 {
		return nil, fmt.Errorf("paper dimensions must be positive")
	}

	inputSource := ""
	duplex := ""
	switch options.Source {
	case "Platen":
		inputSource = "Platen"
	case "ADF Simplex":
		inputSource = "Feeder"
		duplex = "false"
	case "ADF Duplex":
		inputSource = "Feeder"
		duplex = "true"
	default:
		return nil, fmt.Errorf("unsupported scan source %q", options.Source)
	}

	var mode strings.Builder
	if err := xml.EscapeText(&mode, []byte(options.Mode)); err != nil {
		return nil, fmt.Errorf("escape scan mode: %w", err)
	}

	width := toESCLUnits(options.Paper.WidthMicrometres)
	height := toESCLUnits(options.Paper.HeightMicrometres)

	var request strings.Builder
	fmt.Fprintf(&request, `<?xml version="1.0" encoding="UTF-8"?>
<scan:ScanSettings xmlns:scan="http://schemas.hp.com/imaging/escl/2011/05/03" xmlns:escl="http://schemas.hp.com/imaging/escl/2011/05/03" xmlns:pwg="http://www.pwg.org/schemas/2010/12/sm">
  <pwg:Version>2.0</pwg:Version>
  <pwg:ScanRegions>
    <pwg:ScanRegion>
      <pwg:ContentRegionUnits>escl:ThreeHundredthsOfInches</pwg:ContentRegionUnits>
      <pwg:XOffset>0</pwg:XOffset>
      <pwg:YOffset>0</pwg:YOffset>
      <pwg:Width>%d</pwg:Width>
      <pwg:Height>%d</pwg:Height>
    </pwg:ScanRegion>
  </pwg:ScanRegions>
  <pwg:InputSource>%s</pwg:InputSource>
  <scan:ColorMode>%s</scan:ColorMode>
  <pwg:DocumentFormat>application/pdf</pwg:DocumentFormat>
  <scan:XResolution>%d</scan:XResolution>
  <scan:YResolution>%d</scan:YResolution>`, width, height, inputSource, mode.String(), options.Resolution, options.Resolution)
	if duplex != "" {
		fmt.Fprintf(&request, "\n  <scan:Duplex>%s</scan:Duplex>", duplex)
	}
	request.WriteString("\n</scan:ScanSettings>")

	return []byte(request.String()), nil
}
