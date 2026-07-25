//go:build linux

package scanner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func Discover(ctx context.Context) ([]Scanner, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	command := exec.CommandContext(ctx, "scanimage", "-f", "%d%n")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return nil, fmt.Errorf("run scanimage: %w", err)
		}
		return nil, fmt.Errorf("run scanimage: %w: %s", err, message)
	}

	var scanners []Scanner
	for line := range strings.SplitSeq(strings.TrimSpace(stdout.String()), "\n") {
		deviceID := strings.TrimSpace(line)
		if deviceID == "" || strings.HasPrefix(deviceID, "v4l:") {
			continue
		}

		scanners = append(scanners, NewScanner(deviceID))
	}

	return scanners, nil
}
