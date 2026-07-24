package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/slobbe/collate/internal/scanner"
	"github.com/slobbe/collate/internal/utils"
)

// RunScan acquires pages from a scanner and saves them as a PDF.
func RunScan(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return runScan(ctx, args, stdout, stderr, scanner.Discover)
}

func runScan(ctx context.Context, args []string, stdout, stderr io.Writer, discover discoverScanners) int {
	flags := flag.NewFlagSet("collate scan", flag.ContinueOnError)
	flags.SetOutput(stderr)

	deviceFlag := flags.String("device", "", "scanner device identifier")
	outputFlag := flags.String("output", "", "path for scanned PDF")
	sourceFlag := flags.String("source", "", "scanner source, such as ADF")
	batchFlag := flags.Bool("batch", false, "scan all available pages from the source")
	paperFlag := flags.String("paper", "", "paper format: a4")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "usage: %s --device <device> --output <output.pdf> [--source <source> --batch] [--paper a4]\n", flags.Name())
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
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "--device", value: *deviceFlag},
		{name: "--output", value: *outputFlag},
	} {
		if required.value == "" {
			fmt.Fprintf(stderr, "error: %s is required\n", required.name)
			flags.Usage()
			return 2
		}
	}

	if *batchFlag && *sourceFlag == "" {
		fmt.Fprintln(stderr, "error: --source is required when --batch is used")
		flags.Usage()
		return 2
	}

	paper, err := scanner.ParsePaper(*paperFlag)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		flags.Usage()
		return 2
	}

	outputPath, err := utils.NormalizePDFPath(*outputFlag)
	if err != nil {
		fmt.Fprintf(stderr, "error: normalize path %q: %v\n", *outputFlag, err)
		return 2
	}

	scanners, err := discover(ctx)
	if err != nil {
		return reportScanError(stderr, err)
	}

	selected := scannerByDevice(scanners, *deviceFlag)
	if selected == nil {
		fmt.Fprintf(stderr, "error: scanner %q not found; run \"collate scanner list\" to list available scanners\n", *deviceFlag)
		return 2
	}
	defer selected.Close()

	if err := selected.Scan(ctx, scanner.ScanOptions{
		Source: *sourceFlag,
		Batch:  *batchFlag,
		Paper:  paper,
	}); err != nil {
		return reportScanError(stderr, err)
	}
	if err := selected.Save(ctx, outputPath); err != nil {
		return reportScanError(stderr, err)
	}

	fmt.Fprintf(stdout, "scanned successfully: %s\n", outputPath)
	return 0
}

func scannerByDevice(scanners []scanner.Scanner, device string) scanner.Scanner {
	for _, availableScanner := range scanners {
		if availableScanner.Info().Device == device {
			return availableScanner
		}
	}
	return nil
}

func reportScanError(stderr io.Writer, err error) int {
	if errors.Is(err, context.Canceled) {
		fmt.Fprintln(stderr, "interrupted")
		return 130
	}
	fmt.Fprintln(stderr, "error:", err)
	return 1
}
