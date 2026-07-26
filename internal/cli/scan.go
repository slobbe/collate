package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/slobbe/collate/internal/cli/utils"
	"github.com/slobbe/collate/internal/collate"
	"github.com/slobbe/collate/internal/scanner"
	"github.com/slobbe/collate/internal/utils"
)

type startScan func(context.Context, string, scanner.ScanOptions) (*collate.ScanSession, error)
type discoverScannerInfos func(context.Context) ([]scanner.Info, error)
type scannerCapabilities func(context.Context, string) (scanner.Capabilities, error)
type clock func() time.Time

// RunScan interactively acquires front and optional back pages from a scanner and saves a PDF.
func RunScan(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return runScan(ctx, args, stdin, stdout, stderr, collate.StartScanByID, collate.DiscoverScanners, collate.ScannerCapabilities, time.Now)
}

func runScan(
	ctx context.Context,
	args []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	start startScan,
	discover discoverScannerInfos,
	capabilitiesFor scannerCapabilities,
	now clock,
) int {
	flags := flag.NewFlagSet("collate scan", flag.ContinueOnError)
	flags.SetOutput(stderr)
	deviceListFlag := flags.Bool("device-list", false, "list available scanners")
	deviceFlag := flags.String("device", "", "scanner device ID or name")
	sourceFlag := flags.String("source", "", "scanner source ID or name, such as ADF")
	paperFlag := flags.String("paper", "", "paper format: a4, a5, or letter")
	modeFlag := flags.String("mode", "", "scanner color mode ID")
	resolutionFlag := flags.Int("resolution", 0, "scan resolution in DPI")
	outputFlag := flags.String("output", "", "output PDF path")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "usage: %s [--device-list] [--device <device>] [--source <source>] [--paper <paper>] [--mode <mode>] [--resolution <dpi>] [--output <path>]\n", flags.Name())
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		flags.Usage()
		return 2
	}
	if len(flags.Args()) != 0 {
		fmt.Fprintf(stderr, "error: unexpected positional arguments: %v\n", flags.Args())
		flags.Usage()
		return 2
	}
	if *deviceListFlag {
		if *deviceFlag != "" || *sourceFlag != "" || *paperFlag != "" || *modeFlag != "" || *resolutionFlag != 0 || *outputFlag != "" {
			fmt.Fprintln(stderr, "error: --device-list cannot be combined with scan options")
			flags.Usage()
			return 2
		}
		return runDeviceList(ctx, stdout, stderr, discover)
	}

	input := bufio.NewReader(stdin)
	var infos []scanner.Info
	err := (cliutils.Waiting{Message: "Discovering scanners"}).Run(ctx, stdout, func(ctx context.Context) (string, error) {
		var discoverErr error
		infos, discoverErr = discover(ctx)
		return "Discovered scanners", discoverErr
	})
	if err != nil {
		return reportScanError(stderr, err)
	}
	device, err := resolveScanner(ctx, input, stdout, infos, *deviceFlag)
	if err != nil {
		return reportScanError(stderr, err)
	}

	var capabilities scanner.Capabilities
	err = (cliutils.Waiting{Message: "Loading scanner capabilities"}).Run(ctx, stdout, func(ctx context.Context) (string, error) {
		var capabilitiesErr error
		capabilities, capabilitiesErr = capabilitiesFor(ctx, device.ID)
		return "Loaded scanner capabilities", capabilitiesErr
	})
	if err != nil {
		return reportScanError(stderr, err)
	}
	options, err := resolveScanOptions(ctx, input, stdout, capabilities, *sourceFlag, *paperFlag, *modeFlag, *resolutionFlag)
	if err != nil {
		return reportScanError(stderr, err)
	}

	if err := (cliutils.Confirm{Prompt: "Load the front pages", Done: "Ready"}).Run(ctx, input, stdout); err != nil {
		return reportScanError(stderr, err)
	}
	var session *collate.ScanSession
	err = (cliutils.Waiting{Message: "Scanning front pages"}).Run(ctx, stdout, func(ctx context.Context) (string, error) {
		var scanErr error
		session, scanErr = start(ctx, device.ID, options)
		return "Scanned front pages", scanErr
	})
	if err != nil {
		return reportScanError(stderr, err)
	}
	defer session.Close()

	scanBack, err := (cliutils.YesNo{Prompt: "Scan back pages too?"}).Run(ctx, input, stdout)
	if err != nil {
		return reportScanError(stderr, err)
	}

	backOrder := collate.BackOrderReverse
	if scanBack {
		backOrder, err = promptBackOrder(ctx, input, stdout)
		if err != nil {
			return reportScanError(stderr, err)
		}
		if err := (cliutils.Confirm{Prompt: "Load the back pages", Done: "Ready"}).Run(ctx, input, stdout); err != nil {
			return reportScanError(stderr, err)
		}
		if err := (cliutils.Waiting{Message: "Scanning back pages"}).Run(ctx, stdout, func(ctx context.Context) (string, error) {
			return "Scanned back pages", session.ScanBack(ctx)
		}); err != nil {
			return reportScanError(stderr, err)
		}
	}

	var outputPath string
	if *outputFlag == "" {
		outputPath, err = promptOutputPath(ctx, input, stdout, now())
	} else {
		outputPath, err = utils.NormalizePDFPath(*outputFlag)
	}
	if err != nil {
		return reportScanError(stderr, err)
	}
	err = (cliutils.Waiting{Message: "Saving PDF"}).Run(ctx, stdout, func(ctx context.Context) (string, error) {
		if scanBack {
			err := session.Collate(ctx, outputPath, backOrder)
			return fmt.Sprintf("Scanned and collated successfully: %s", outputPath), err
		}
		err := session.SaveFront(ctx, outputPath)
		return fmt.Sprintf("Scanned successfully: %s", outputPath), err
	})
	if err != nil {
		return reportScanError(stderr, err)
	}
	return 0
}

func runDeviceList(ctx context.Context, stdout, stderr io.Writer, discover discoverScannerInfos) int {
	infos, err := discover(ctx)
	if err != nil {
		return reportScanError(stderr, err)
	}
	if len(infos) == 0 {
		fmt.Fprintln(stdout, "No scanners found.")
		return 0
	}

	fmt.Fprintln(stdout, "Available scanners:")
	for _, info := range infos {
		if err := ctx.Err(); err != nil {
			return reportScanError(stderr, err)
		}
		fmt.Fprintf(stdout, "- %s\n  device: %s\n", scannerName(info), info.ID)
	}

	return 0
}

func reportScanError(stderr io.Writer, err error) int {
	if errors.Is(err, context.Canceled) {
		fmt.Fprintln(stderr, "interrupted")
		return 130
	}
	fmt.Fprintln(stderr, "error:", err)
	return 1
}
