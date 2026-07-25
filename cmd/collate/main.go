package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/slobbe/collate/internal/cli"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	go func() {
		<-ctx.Done()
		stop()
	}()

	exitCode := cli.Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr, version)
	stop()
	os.Exit(exitCode)
}
