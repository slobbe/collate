//go:build linux

package scanner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/slobbe/collate/internal/pdf"
)

var scanimageDeviceLine = regexp.MustCompile("^device [`']([^`']+)[`'] is (.+)$")

type listCommand func(context.Context) ([]byte, []byte, error)
type scanCommand func(context.Context, string, ScanOptions, string) error
type pdfWriter func([]string, string, Paper) error

type linuxScanner struct {
	info       Info
	scan       scanCommand
	writePDF   pdfWriter
	createTemp func() (string, error)
	workDir    string
	pages      []string
	paper      Paper
}

func newLinuxScanner(info Info) *linuxScanner {
	return &linuxScanner{
		info:     info,
		scan:     scanWithScanimage,
		writePDF: writeScannedPDF,
		createTemp: func() (string, error) {
			return os.MkdirTemp("", "collate-scan-")
		},
	}
}

func (scanner *linuxScanner) Info() Info {
	return scanner.info
}

// Scan acquires one or more pages from the scanner into temporary files.
func (scanner *linuxScanner) Scan(ctx context.Context, options ScanOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if options.Paper != "" && options.Paper != PaperA4 {
		return fmt.Errorf("unsupported paper format %q", options.Paper)
	}
	if err := scanner.Close(); err != nil {
		return err
	}

	workDir, err := scanner.createTemp()
	if err != nil {
		return fmt.Errorf("create scan workspace: %w", err)
	}

	if err := scanner.scan(ctx, scanner.info.Device, options, workDir); err != nil {
		os.RemoveAll(workDir)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("scan device %q: %w", scanner.info.Device, err)
	}

	pages, err := filepath.Glob(filepath.Join(workDir, "page-*.png"))
	if err != nil {
		os.RemoveAll(workDir)
		return fmt.Errorf("find scanned pages: %w", err)
	}
	sort.Strings(pages)
	if len(pages) == 0 {
		os.RemoveAll(workDir)
		return fmt.Errorf("scan device %q: no pages were produced", scanner.info.Device)
	}

	scanner.workDir = workDir
	scanner.pages = pages
	scanner.paper = options.Paper
	return nil
}

// Save writes the most recently scanned pages as a PDF.
func (scanner *linuxScanner) Save(ctx context.Context, outputPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(scanner.pages) == 0 {
		return fmt.Errorf("no scanned pages to save")
	}

	temporaryDir, err := os.MkdirTemp(filepath.Dir(outputPath), ".collate-scan-")
	if err != nil {
		return fmt.Errorf("create temporary PDF directory: %w", err)
	}
	defer os.RemoveAll(temporaryDir)

	temporaryOutputPath := filepath.Join(temporaryDir, filepath.Base(outputPath))
	if err := scanner.writePDF(scanner.pages, temporaryOutputPath, scanner.paper); err != nil {
		return fmt.Errorf("create PDF from scanned pages: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(temporaryOutputPath, outputPath); err != nil {
		return fmt.Errorf("save scanned PDF: %w", err)
	}

	return nil
}

// Close removes temporary data created by Scan.
func (scanner *linuxScanner) Close() error {
	workDir := scanner.workDir
	scanner.workDir = ""
	scanner.pages = nil
	scanner.paper = ""
	if workDir == "" {
		return nil
	}
	if err := os.RemoveAll(workDir); err != nil {
		return fmt.Errorf("remove scan workspace: %w", err)
	}
	return nil
}

func scanWithScanimage(ctx context.Context, device string, options ScanOptions, workDir string) error {
	path, err := exec.LookPath("scanimage")
	if err != nil {
		return fmt.Errorf("find scanimage (install sane-utils and ensure it is on PATH): %w", err)
	}

	command := exec.CommandContext(ctx, path, scanimageArgs(device, options, workDir)...)
	var output *os.File
	if !options.Batch {
		output, err = os.Create(filepath.Join(workDir, "page-0001.png"))
		if err != nil {
			return fmt.Errorf("create scanned page: %w", err)
		}
		command.Stdout = output
	}

	var stderr bytes.Buffer
	command.Stderr = &stderr
	runErr := command.Run()
	var closeErr error
	if output != nil {
		closeErr = output.Close()
	}
	if runErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return fmt.Errorf("run scanimage: %w", runErr)
		}
		return fmt.Errorf("run scanimage: %w: %s", runErr, message)
	}
	if closeErr != nil {
		return fmt.Errorf("close scanned page: %w", closeErr)
	}

	return nil
}

func scanimageArgs(device string, options ScanOptions, workDir string) []string {
	args := []string{"--device-name", device, "--format=png"}
	if options.Source != "" {
		args = append(args, "--source", options.Source)
	}
	if options.Paper == PaperA4 {
		args = append(args, "-x", "210", "-y", "297")
	}
	if options.Batch {
		args = append(args, "--batch="+filepath.Join(workDir, "page-%04d.png"))
	}
	return args
}

func writeScannedPDF(images []string, outputPath string, paper Paper) error {
	switch paper {
	case "":
		return pdf.ImportImages(images, outputPath)
	case PaperA4:
		return pdf.ImportImagesA4(images, outputPath)
	default:
		return fmt.Errorf("unsupported paper format %q", paper)
	}
}

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
		scanners = append(scanners, newLinuxScanner(Info{
			Device:      matches[1],
			Description: description,
		}))
	}

	return scanners
}
