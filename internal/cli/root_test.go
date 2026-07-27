package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunRootUsage(t *testing.T) {
	tests := []struct {
		name string
		args []string
		code int
		want string
	}{
		{
			name: "no command",
			code: 2,
			want: "usage: collate <command> [flags]",
		},
		{
			name: "unknown command",
			args: []string{"split"},
			code: 2,
			want: `error: unknown command "split"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, _, stderr := runRootCommand(test.args)
			if code != test.code {
				t.Fatalf("Run() exit code = %d, want %d", code, test.code)
			}
			if !strings.Contains(stderr, test.want) {
				t.Fatalf("stderr = %q, want %q", stderr, test.want)
			}
		})
	}
}

func TestRunVersion(t *testing.T) {
	code, stdout, stderr := runRootCommand([]string{"--version"})

	if code != 0 {
		t.Fatalf("Run() exit code = %d, want 0", code)
	}
	const want = "collate dev\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunReportsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	code, stdout, stderr := runRootCommandWithContext(ctx, []string{"merge"})
	if code != 130 {
		t.Fatalf("Run() exit code = %d, want 130", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if stderr != "interrupted\n" {
		t.Fatalf("stderr = %q, want interruption message", stderr)
	}
}

func runRootCommand(args []string) (int, string, string) {
	return runRootCommandWithContext(context.Background(), args)
}

func runRootCommandWithContext(ctx context.Context, args []string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := Run(ctx, args, strings.NewReader(""), &stdout, &stderr, nil, "dev")
	return code, stdout.String(), stderr.String()
}
