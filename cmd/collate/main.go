package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/slobbe/collate/internal/collate"
	"github.com/slobbe/collate/internal/utils"
)

func main() {
	backOrderFlag := flag.String(
		"back-order",
		string(collate.BackOrderReverse),
		"order of pages in back.pdf: reverse or forward",
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [flags] <front.pdf> <back.pdf> <output.pdf>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(flag.Args()) != 3 {
		flag.Usage()
		os.Exit(2)
	}

	backOrder, err := collate.ParseBackOrder(*backOrderFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		flag.Usage()
		os.Exit(2)
	}

	paths := make([]string, len(flag.Args()))
	for i, path := range flag.Args() {
		paths[i], err = utils.NormalizePDFPath(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: normalize path %q: %v\n", path, err)
			os.Exit(2)
		}
	}

	if err := collate.Collate(paths[0], paths[1], paths[2], backOrder); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Printf("collated successfully: %s\n", paths[2])
	os.Exit(0)
}
