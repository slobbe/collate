//go:build linux

package scanner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var scanimageDeviceLine = regexp.MustCompile("^device [`']([^`']+)[`'] is (.+)$")

type listCommand func(context.Context) ([]byte, []byte, error)

// Discover returns the scanners SANE reports as available on Linux.
func Discover(ctx context.Context) ([]Scanner, error) {
	return discover(ctx, listScanimage)
}

func discover(ctx context.Context, list listCommand) ([]Scanner, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	output, diagnostics, err := list(ctx)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		message := strings.TrimSpace(string(diagnostics))
		if message == "" {
			return nil, fmt.Errorf("run scanimage -L: %w", err)
		}
		return nil, fmt.Errorf("run scanimage -L: %w: %s", err, message)
	}

	return parseScanimageDevices(output), nil
}

func listScanimage(ctx context.Context) ([]byte, []byte, error) {
	path, err := exec.LookPath("scanimage")
	if err != nil {
		return nil, nil, fmt.Errorf("find scanimage (install sane-utils and ensure it is on PATH): %w", err)
	}

	command := exec.CommandContext(ctx, path, "-L")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func parseScanimageDevices(output []byte) []Scanner {
	var scanners []Scanner

	for _, line := range strings.Split(string(output), "\n") {
		matches := scanimageDeviceLine.FindStringSubmatch(strings.TrimSpace(line))
		if matches == nil {
			continue
		}

		description := strings.TrimPrefix(matches[2], "a ")
		scanners = append(scanners, discoveredScanner{
			info: Info{
				Device:      matches[1],
				Description: description,
			},
		})
	}

	return scanners
}
