package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/slobbe/collate/internal/collate"
	"github.com/slobbe/collate/internal/scanner"
	"github.com/slobbe/collate/internal/utils"
)

type discoverScanners func(context.Context) ([]scanner.Scanner, error)
type mergeScans func(context.Context, string, string, string, collate.BackOrder) error

// RunScan acquires front and optional back pages from a scanner and saves a PDF.
func RunScan(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return runScan(ctx, args, stdin, stdout, stderr, scanner.DiscoverLinux, collate.Collate)
}

func runScan(
	ctx context.Context,
	args []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	discover discoverScanners,
	merge mergeScans,
) int {
	flags := flag.NewFlagSet("collate scan", flag.ContinueOnError)
	flags.SetOutput(stderr)

	deviceFlag := flags.String("device", "", "scanner device identifier")
	outputFlag := flags.String("output", "", "path for scanned PDF")
	sourceFlag := flags.String("source", "", "scanner source from scanner capabilities")
	paperFlag := flags.String("paper", "", "paper format: a4, a5, or letter")
	deviceListFlag := flags.Bool("device-list", false, "list available scanners")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "usage: %s --device-list | --device <device> --output <output.pdf> [--source <source>] [--paper <paper>]\n", flags.Name())
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
		if *deviceFlag != "" || *outputFlag != "" || *sourceFlag != "" || *paperFlag != "" {
			fmt.Fprintln(stderr, "error: --device-list cannot be combined with scan options")
			flags.Usage()
			return 2
		}
		return runDeviceList(ctx, stdout, stderr, discover)
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

	var paper scanner.Paper
	if *paperFlag != "" {
		parsedPaper, err := scanner.ParsePaper(*paperFlag)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			flags.Usage()
			return 2
		}
		paper = parsedPaper
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
		fmt.Fprintf(stderr, "error: scanner %q not found; run \"collate scan --device-list\" to list available scanners\n", *deviceFlag)
		return 2
	}
	defer selected.Close(context.Background())

	options := scanner.ScanOptions{
		Source: scanner.Source(*sourceFlag),
		Paper:  paper,
	}
	input := bufio.NewReader(stdin)

	if err := waitForScanStart(input, stdout, "Load the front pages."); err != nil {
		return reportScanError(stderr, err)
	}
	if err := selected.Scan(ctx, options); err != nil {
		return reportScanError(stderr, err)
	}

	scanBack, err := promptYesNo(input, stdout, "Scan back pages too? [y/N]: ")
	if err != nil {
		return reportScanError(stderr, err)
	}
	if !scanBack {
		if err := selected.Save(ctx, outputPath); err != nil {
			return reportScanError(stderr, err)
		}
		fmt.Fprintf(stdout, "scanned successfully: %s\n", outputPath)
		return 0
	}

	backOrder, err := promptBackOrder(input, stdout)
	if err != nil {
		return reportScanError(stderr, err)
	}

	temporaryDir, err := os.MkdirTemp(filepath.Dir(outputPath), ".collate-duplex-")
	if err != nil {
		return reportScanError(stderr, fmt.Errorf("create temporary scan directory: %w", err))
	}
	defer os.RemoveAll(temporaryDir)

	frontPath := filepath.Join(temporaryDir, "front.pdf")
	if err := selected.Save(ctx, frontPath); err != nil {
		return reportScanError(stderr, err)
	}

	if err := waitForScanStart(input, stdout, "Load the back pages."); err != nil {
		return reportScanError(stderr, err)
	}
	if err := selected.Scan(ctx, options); err != nil {
		return reportScanError(stderr, err)
	}

	backPath := filepath.Join(temporaryDir, "back.pdf")
	if err := selected.Save(ctx, backPath); err != nil {
		return reportScanError(stderr, err)
	}
	if err := merge(ctx, frontPath, backPath, outputPath, backOrder); err != nil {
		return reportScanError(stderr, err)
	}

	fmt.Fprintf(stdout, "scanned and collated successfully: %s\n", outputPath)
	return 0
}

func waitForScanStart(input *bufio.Reader, output io.Writer, pages string) error {
	fmt.Fprintf(output, "%s Press Enter to start scanning.\n", pages)
	_, err := readPrompt(input)
	if err != nil {
		return fmt.Errorf("read scan confirmation: %w", err)
	}
	return nil
}

func promptYesNo(input *bufio.Reader, output io.Writer, prompt string) (bool, error) {
	for {
		fmt.Fprint(output, prompt)
		answer, err := readPrompt(input)
		if err != nil {
			return false, fmt.Errorf("read response: %w", err)
		}

		switch strings.ToLower(answer) {
		case "", "n", "no":
			return false, nil
		case "y", "yes":
			return true, nil
		default:
			fmt.Fprintln(output, "Please answer yes or no.")
		}
	}
}

func promptBackOrder(input *bufio.Reader, output io.Writer) (collate.BackOrder, error) {
	for {
		fmt.Fprint(output, "Back-page order [reverse/forward] (reverse): ")
		answer, err := readPrompt(input)
		if err != nil {
			return "", fmt.Errorf("read back-page order: %w", err)
		}
		if answer == "" {
			return collate.BackOrderReverse, nil
		}

		order, err := collate.ParseBackOrder(strings.ToLower(answer))
		if err == nil {
			return order, nil
		}
		fmt.Fprintln(output, "Please enter reverse or forward.")
	}
}

func readPrompt(input *bufio.Reader) (string, error) {
	line, err := input.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if err == io.EOF && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func runDeviceList(ctx context.Context, stdout, stderr io.Writer, discover discoverScanners) int {
	scanners, err := discover(ctx)
	if err != nil {
		return reportScanError(stderr, err)
	}
	if len(scanners) == 0 {
		fmt.Fprintln(stdout, "No scanners found.")
		return 0
	}

	fmt.Fprintln(stdout, "Available scanners:")
	for _, availableScanner := range scanners {
		if err := ctx.Err(); err != nil {
			return reportScanError(stderr, err)
		}
		fmt.Fprintf(stdout, "- %s\n", availableScanner.ID())
	}

	return 0
}

func scannerByDevice(scanners []scanner.Scanner, device string) scanner.Scanner {
	for _, availableScanner := range scanners {
		if availableScanner.ID() == device {
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
