//go:build !linux

package scanner

import (
	"context"
	"fmt"
	"runtime"
)

// Discover reports that scanner discovery has not yet been implemented on this platform.
func Discover(ctx context.Context) ([]Scanner, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("scanner discovery is not supported on %s", runtime.GOOS)
}
