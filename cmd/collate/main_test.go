package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRejectsMissingRequiredInputFlags(t *testing.T) {
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
			code, _, stderr := runCommand(test.args)
			if code != 2 {
				t.Fatalf("run() exit code = %d, want 2", code)
			}
			if !strings.Contains(stderr, test.want) {
				t.Fatalf("stderr = %q, want %q", stderr, test.want)
			}
			if !strings.Contains(stderr, "usage: collate -f <front.pdf> -b <back.pdf> -o <output.pdf> [flags]") {
				t.Fatalf("stderr = %q, want usage", stderr)
			}
		})
	}
}

func TestRunRejectsPositionalArguments(t *testing.T) {
	code, _, stderr := runCommand([]string{
		"-f", "front.pdf",
		"-b", "back.pdf",
		"-o", "output.pdf",
		"extra.pdf",
	})

	if code != 2 {
		t.Fatalf("run() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "error: unexpected positional arguments: [extra.pdf]") {
		t.Fatalf("stderr = %q, want positional-argument error", stderr)
	}
}

func TestRunAcceptsNewFlagsWithBackOrder(t *testing.T) {
	code, _, stderr := runCommand([]string{
		"-f", "front.txt",
		"-b", "back.pdf",
		"-o", "output.pdf",
		"-back-order=forward",
	})

	if code != 2 {
		t.Fatalf("run() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, `error: normalize path "front.txt": expected a .pdf file`) {
		t.Fatalf("stderr = %q, want front path validation error", stderr)
	}
}

func TestRunRejectsInvalidBackOrder(t *testing.T) {
	code, _, stderr := runCommand([]string{
		"-f", "front.pdf",
		"-b", "back.pdf",
		"-o", "output.pdf",
		"-back-order=sideways",
	})

	if code != 2 {
		t.Fatalf("run() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, `error: invalid back order "sideways"`) {
		t.Fatalf("stderr = %q, want back-order error", stderr)
	}
}

func TestRunHelp(t *testing.T) {
	code, _, stderr := runCommand([]string{"-h"})

	if code != 0 {
		t.Fatalf("run() exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr, "usage: collate -f <front.pdf> -b <back.pdf> -o <output.pdf> [flags]") {
		t.Fatalf("stderr = %q, want usage", stderr)
	}
}

func runCommand(args []string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}
