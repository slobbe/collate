package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/slobbe/collate/internal/collate"
	"github.com/slobbe/collate/internal/utils"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		rootUsage(stderr)
		return 2
	}

	switch args[0] {
	case "merge":
		return runMerge(args[1:], stdout, stderr)
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
	fmt.Fprintln(output, "\ncommands:")
	fmt.Fprintln(output, "  merge    rebuild a duplex PDF from front and back simplex scans")
}

func runMerge(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("collate merge", flag.ContinueOnError)
	flags.SetOutput(stderr)

	frontFlag := flags.String("f", "", "path to front PDF")
	backFlag := flags.String("b", "", "path to back PDF")
	outputFlag := flags.String("o", "", "path for output PDF")
	var backOrderValue string
	flags.StringVar(
		&backOrderValue,
		"bo",
		string(collate.BackOrderReverse),
		"page order of back PDF relative to front PDF: reverse or forward",
	)
	flags.StringVar(
		&backOrderValue,
		"backorder",
		string(collate.BackOrderReverse),
		"page order of back PDF relative to front PDF: reverse or forward",
	)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "usage: %s -f <front.pdf> -b <back.pdf> -o <output.pdf> [flags]\n", flags.Name())
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
		{name: "-f", value: *frontFlag},
		{name: "-b", value: *backFlag},
		{name: "-o", value: *outputFlag},
	} {
		if required.value == "" {
			fmt.Fprintf(stderr, "error: %s is required\n", required.name)
			flags.Usage()
			return 2
		}
	}

	backOrder, err := collate.ParseBackOrder(backOrderValue)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		flags.Usage()
		return 2
	}

	frontPath, err := utils.NormalizePDFPath(*frontFlag)
	if err != nil {
		fmt.Fprintf(stderr, "error: normalize path %q: %v\n", *frontFlag, err)
		return 2
	}
	backPath, err := utils.NormalizePDFPath(*backFlag)
	if err != nil {
		fmt.Fprintf(stderr, "error: normalize path %q: %v\n", *backFlag, err)
		return 2
	}
	outputPath, err := utils.NormalizePDFPath(*outputFlag)
	if err != nil {
		fmt.Fprintf(stderr, "error: normalize path %q: %v\n", *outputFlag, err)
		return 2
	}

	if err := collate.Collate(frontPath, backPath, outputPath, backOrder); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	fmt.Fprintf(stdout, "collated successfully: %s\n", outputPath)
	return 0
}
