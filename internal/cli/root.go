package cli

import (
	"context"
	"fmt"
	"io"
)

// Run executes a collate command and returns its process exit code.
func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, version string) int {
	if err := ctx.Err(); err != nil {
		fmt.Fprintln(stderr, "interrupted")
		return 130
	}

	if len(args) == 0 {
		rootUsage(stderr)
		return 2
	}

	switch args[0] {
	case "merge":
		return RunMerge(ctx, args[1:], stdout, stderr)
	case "scan":
		return RunScan(ctx, args[1:], stdin, stdout, stderr)
	case "--version":
		fmt.Fprintf(stdout, "collate %s\n", version)
		return 0
	case "-h", "--help", "help":
		rootUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n", args[0])
		rootUsage(stderr)
		return 2
	}
}

func rootUsage(output io.Writer) {
	fmt.Fprintln(output, "usage: collate <command> [flags]")
	fmt.Fprintln(output, "\nflags:")
	fmt.Fprintln(output, "  --version  print the collate version")
	fmt.Fprintln(output, "\ncommands:")
	fmt.Fprintln(output, "  merge    rebuild a duplex PDF from front and back simplex scans")
	fmt.Fprintln(output, "  scan     scan pages to a PDF or list scanners")
}
