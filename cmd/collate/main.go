package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/slobbe/collate/internal/cli"
	"github.com/slobbe/collate/internal/collate"
	"github.com/slobbe/collate/internal/scanner/escl"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	go func() {
		<-ctx.Done()
		stop()
	}()

	scans := collate.NewScanService(escl.NewProvider())
	exitCode := cli.Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr, scans, version)
	stop()
	os.Exit(exitCode)
}
