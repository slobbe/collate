package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

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
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "usage: %s [--device-list]\n", flags.Name())
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
		return runDeviceList(ctx, stdout, stderr, discover)
	}

	input := bufio.NewReader(stdin)
	infos, err := discover(ctx)
	if err != nil {
		return reportScanError(stderr, err)
	}
	device, err := selectScanner(input, stdout, infos)
	if err != nil {
		return reportScanError(stderr, err)
	}

	capabilities, err := capabilitiesFor(ctx, device.ID)
	if err != nil {
		return reportScanError(stderr, err)
	}
	options, err := selectScanOptions(input, stdout, capabilities)
	if err != nil {
		return reportScanError(stderr, err)
	}

	if err := waitForScanStart(input, stdout, "Load the front pages."); err != nil {
		return reportScanError(stderr, err)
	}
	session, err := start(ctx, device.ID, options)
	if err != nil {
		return reportScanError(stderr, err)
	}
	defer session.Close()

	scanBack, err := promptYesNo(input, stdout, "Scan back pages too? [y/N]: ")
	if err != nil {
		return reportScanError(stderr, err)
	}

	backOrder := collate.BackOrderReverse
	if scanBack {
		backOrder, err = promptBackOrder(input, stdout)
		if err != nil {
			return reportScanError(stderr, err)
		}
		if err := waitForScanStart(input, stdout, "Load the back pages."); err != nil {
			return reportScanError(stderr, err)
		}
		if err := session.ScanBack(ctx); err != nil {
			return reportScanError(stderr, err)
		}
	}

	outputPath, err := promptOutputPath(input, stdout, now())
	if err != nil {
		return reportScanError(stderr, err)
	}
	if scanBack {
		err = session.Collate(ctx, outputPath, backOrder)
	} else {
		err = session.SaveFront(ctx, outputPath)
	}
	if err != nil {
		return reportScanError(stderr, err)
	}

	if scanBack {
		fmt.Fprintf(stdout, "scanned and collated successfully: %s\n", outputPath)
	} else {
		fmt.Fprintf(stdout, "scanned successfully: %s\n", outputPath)
	}
	return 0
}

func selectScanner(input *bufio.Reader, output io.Writer, infos []scanner.Info) (scanner.Info, error) {
	if len(infos) == 0 {
		return scanner.Info{}, fmt.Errorf("no scanners found")
	}
	if len(infos) == 1 {
		name := scannerName(infos[0])
		fmt.Fprintf(output, "Using scanner: %s\n", name)
		return infos[0], nil
	}

	fmt.Fprintln(output, "Available scanners:")
	for index, info := range infos {
		fmt.Fprintf(output, "  %d. %s\n", index+1, scannerName(info))
	}
	index, err := promptIndex(input, output, "Select scanner", len(infos))
	if err != nil {
		return scanner.Info{}, err
	}
	return infos[index], nil
}

func selectScanOptions(input *bufio.Reader, output io.Writer, capabilities scanner.Capabilities) (scanner.ScanOptions, error) {
	source, err := selectSource(input, output, capabilities.Sources)
	if err != nil {
		return scanner.ScanOptions{}, err
	}
	paper, err := promptPaper(input, output)
	if err != nil {
		return scanner.ScanOptions{}, err
	}
	mode, err := selectMode(input, output, source.ColorModes)
	if err != nil {
		return scanner.ScanOptions{}, err
	}
	resolution, err := selectResolution(input, output, source.Resolutions)
	if err != nil {
		return scanner.ScanOptions{}, err
	}
	return scanner.ScanOptions{Source: source.ID, Paper: paper, Mode: mode, Resolution: resolution}, nil
}

func selectSource(input *bufio.Reader, output io.Writer, sources []scanner.Source) (scanner.Source, error) {
	if len(sources) == 0 {
		return scanner.Source{}, fmt.Errorf("scanner does not advertise scan sources")
	}
	fmt.Fprintln(output, "Available sources:")
	for index, source := range sources {
		name := source.Name
		if name == "" {
			name = source.ID
		}
		fmt.Fprintf(output, "  %d. %s\n", index+1, name)
	}
	index, err := promptIndex(input, output, "Select source", len(sources))
	if err != nil {
		return scanner.Source{}, err
	}
	return sources[index], nil
}

func promptPaper(input *bufio.Reader, output io.Writer) (scanner.Paper, error) {
	for {
		fmt.Fprint(output, "Paper [a4/a5/letter] (a4): ")
		value, err := readPrompt(input)
		if err != nil {
			return scanner.Paper{}, fmt.Errorf("read paper format: %w", err)
		}
		if value == "" {
			return scanner.PaperA4, nil
		}
		paper, err := parsePaper(value)
		if err == nil {
			return paper, nil
		}
		fmt.Fprintln(output, "Please enter a4, a5, or letter.")
	}
}

func selectMode(input *bufio.Reader, output io.Writer, modes []string) (string, error) {
	if len(modes) == 0 {
		return "", fmt.Errorf("scanner does not advertise color modes")
	}
	fmt.Fprintln(output, "Available color modes:")
	for index, mode := range modes {
		fmt.Fprintf(output, "  %d. %s\n", index+1, mode)
	}
	index, err := promptIndex(input, output, "Select color mode", len(modes))
	if err != nil {
		return "", err
	}
	return modes[index], nil
}

func selectResolution(input *bufio.Reader, output io.Writer, resolutions []int) (int, error) {
	if len(resolutions) == 0 {
		return 0, fmt.Errorf("scanner does not advertise scan resolutions")
	}
	fmt.Fprintln(output, "Available resolutions:")
	for index, resolution := range resolutions {
		fmt.Fprintf(output, "  %d. %d DPI\n", index+1, resolution)
	}
	index, err := promptIndex(input, output, "Select resolution", len(resolutions))
	if err != nil {
		return 0, err
	}
	return resolutions[index], nil
}

func promptIndex(input *bufio.Reader, output io.Writer, label string, count int) (int, error) {
	for {
		fmt.Fprintf(output, "%s [1]: ", label)
		value, err := readPrompt(input)
		if err != nil {
			return 0, fmt.Errorf("read selection: %w", err)
		}
		if value == "" {
			return 0, nil
		}
		index, err := strconv.Atoi(value)
		if err == nil && index >= 1 && index <= count {
			return index - 1, nil
		}
		fmt.Fprintf(output, "Please enter a number from 1 to %d.\n", count)
	}
}

func promptOutputPath(input *bufio.Reader, output io.Writer, now time.Time) (string, error) {
	defaultPath := now.Format("scan_20060102_150405.pdf")
	for {
		fmt.Fprintf(output, "Output path [%s]: ", defaultPath)
		path, err := readPrompt(input)
		if err != nil {
			return "", fmt.Errorf("read output path: %w", err)
		}
		if path == "" {
			path = defaultPath
		}
		outputPath, err := utils.NormalizePDFPath(path)
		if err == nil {
			return outputPath, nil
		}
		fmt.Fprintf(output, "Invalid output path: %v\n", err)
	}
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

func scannerName(info scanner.Info) string {
	if info.Name != "" {
		return info.Name
	}
	return info.ID
}

func parsePaper(value string) (scanner.Paper, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "a4", "din-a4":
		return scanner.PaperA4, nil
	case "a5", "din-a5":
		return scanner.PaperA5, nil
	case "letter":
		return scanner.PaperLetter, nil
	default:
		return scanner.Paper{}, fmt.Errorf("invalid paper format %q", value)
	}
}

func reportScanError(stderr io.Writer, err error) int {
	if errors.Is(err, context.Canceled) {
		fmt.Fprintln(stderr, "interrupted")
		return 130
	}
	fmt.Fprintln(stderr, "error:", err)
	return 1
}
