package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunMergeRejectsMissingRequiredInputFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "front",
			args: []string{"-b", "back.pdf", "-o", "output.pdf"},
			want: "error: -f is required",
		},
		{
			name: "back",
			args: []string{"-f", "front.pdf", "-o", "output.pdf"},
			want: "error: -b is required",
		},
		{
			name: "output",
			args: []string{"-f", "front.pdf", "-b", "back.pdf"},
			want: "error: -o is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, _, stderr := runMergeCommand(test.args)
			if code != 2 {
				t.Fatalf("RunMerge() exit code = %d, want 2", code)
			}
			if !strings.Contains(stderr, test.want) {
				t.Fatalf("stderr = %q, want %q", stderr, test.want)
			}
			if !strings.Contains(stderr, "usage: collate merge -f <front.pdf> -b <back.pdf> -o <output.pdf> [flags]") {
				t.Fatalf("stderr = %q, want usage", stderr)
			}
		})
	}
}

func TestRunMergeRejectsPositionalArguments(t *testing.T) {
	code, _, stderr := runMergeCommand([]string{
		"-f", "front.pdf",
		"-b", "back.pdf",
		"-o", "output.pdf",
		"extra.pdf",
	})

	if code != 2 {
		t.Fatalf("RunMerge() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "error: unexpected positional arguments: [extra.pdf]") {
		t.Fatalf("stderr = %q, want positional-argument error", stderr)
	}
}

func TestRunMergeAcceptsBackOrder(t *testing.T) {
	code, _, stderr := runMergeCommand([]string{
		"-f", "front.txt",
		"-b", "back.pdf",
		"-o", "output.pdf",
		"--backorder=forward",
	})

	if code != 2 {
		t.Fatalf("RunMerge() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, `error: normalize path "front.txt": expected a .pdf file`) {
		t.Fatalf("stderr = %q, want front path validation error", stderr)
	}
}

func TestRunMergeRejectsInvalidBackOrder(t *testing.T) {
	code, _, stderr := runMergeCommand([]string{
		"-f", "front.pdf",
		"-b", "back.pdf",
		"-o", "output.pdf",
		"--backorder=sideways",
	})

	if code != 2 {
		t.Fatalf("RunMerge() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, `error: invalid back order "sideways"`) {
		t.Fatalf("stderr = %q, want back-order error", stderr)
	}
}

func TestRunMergeHelp(t *testing.T) {
	code, _, stderr := runMergeCommand([]string{"-h"})

	if code != 0 {
		t.Fatalf("RunMerge() exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr, "usage: collate merge -f <front.pdf> -b <back.pdf> -o <output.pdf> [flags]") {
		t.Fatalf("stderr = %q, want usage", stderr)
	}
}

func runMergeCommand(args []string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := RunMerge(context.Background(), args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}
