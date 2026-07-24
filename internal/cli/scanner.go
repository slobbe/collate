package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/slobbe/collate/internal/scanner"
)

type discoverScanners func(context.Context) ([]scanner.Scanner, error)

// RunScanner executes a scanner command.
func RunScanner(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return runScanner(ctx, args, stdout, stderr, scanner.Discover)
}

func runScanner(ctx context.Context, args []string, stdout, stderr io.Writer, discover discoverScanners) int {
	if err := ctx.Err(); err != nil {
		fmt.Fprintln(stderr, "interrupted")
		return 130
	}

	if len(args) == 0 {
		scannerUsage(stderr)
		return 2
	}

	switch args[0] {
	case "list":
		return runScannerList(ctx, args[1:], stdout, stderr, discover)
	case "-h", "--help", "help":
		scannerUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "error: unknown scanner command %q\n", args[0])
		scannerUsage(stderr)
		return 2
	}
}

func scannerUsage(output io.Writer) {
	fmt.Fprintln(output, "usage: collate scanner <command>")
	fmt.Fprintln(output, "\ncommands:")
	fmt.Fprintln(output, "  list  list available scanners")
}

func runScannerList(ctx context.Context, args []string, stdout, stderr io.Writer, discover discoverScanners) int {
	flags := flag.NewFlagSet("collate scanner list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "usage: %s\n", flags.Name())
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

	scanners, err := discover(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(stderr, "interrupted")
			return 130
		}
		fmt.Fprintln(stderr, "error: discover scanners:", err)
		return 1
	}

	if len(scanners) == 0 {
		fmt.Fprintln(stdout, "No scanners found.")
		return 0
	}

	fmt.Fprintln(stdout, "Available scanners:")
	for _, availableScanner := range scanners {
		if err := ctx.Err(); err != nil {
			fmt.Fprintln(stderr, "interrupted")
			return 130
		}

		info := availableScanner.Info()
		description := info.Description
		if description == "" {
			description = info.Device
		}
		fmt.Fprintf(stdout, "- %s\n  device: %s\n", description, info.Device)
	}

	return 0
}
