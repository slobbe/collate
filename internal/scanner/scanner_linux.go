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
	"strconv"
	"strings"

	"github.com/slobbe/collate/internal/pdf"
)

type LinuxScanner struct {
	deviceID     string
	workingDir   string
	capabilities *Capabilities
}

func NewScanner(deviceID string) *LinuxScanner {
	deviceID = strings.TrimSpace(deviceID)

	if deviceID == "" {
		return nil
	}

	return &LinuxScanner{
		deviceID:     deviceID,
		workingDir:   "",
		capabilities: nil,
	}
}

func (s *LinuxScanner) ID() string {
	return s.deviceID
}

func (s *LinuxScanner) Capabilities(ctx context.Context, force bool) Capabilities {
	if !force && s.capabilities != nil {
		return *s.capabilities
	}
	if ctx.Err() != nil {
		return Capabilities{}
	}

	capabilities, err := getScannerCapabilities(ctx, s.deviceID)
	if err != nil {
		return Capabilities{}
	}

	s.capabilities = &capabilities
	return capabilities
}

func (s *LinuxScanner) Scan(ctx context.Context, options ScanOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if options.Resolution < 0 {
		return fmt.Errorf("resolution must not be negative")
	}

	args := []string{"-d", s.deviceID, "--format=png"}
	if options.Source != "" {
		args = append(args, "--source", string(options.Source))
	}
	switch options.Paper {
	case "":
	case PaperA4:
		args = append(args, "-x", "210", "-y", "297")
	case PaperA5:
		args = append(args, "-x", "148", "-y", "210")
	case PaperLetter:
		args = append(args, "-x", "215.9", "-y", "279.4")
	default:
		return fmt.Errorf("unsupported paper format %q", options.Paper)
	}
	switch options.Mode {
	case "":
	case ModeColor:
		args = append(args, "--mode", "Color")
	case ModeGray:
		args = append(args, "--mode", "Gray")
	default:
		return fmt.Errorf("unsupported scan mode %q", options.Mode)
	}
	if options.Resolution > 0 {
		args = append(args, "--resolution", strconv.Itoa(options.Resolution))
	}

	if s.workingDir != "" {
		if err := os.RemoveAll(s.workingDir); err != nil {
			return fmt.Errorf("remove previous scan workspace: %w", err)
		}
		s.workingDir = ""
	}

	workingDir, err := os.MkdirTemp("", "collate-scan-")
	if err != nil {
		return fmt.Errorf("create scan workspace: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			os.RemoveAll(workingDir)
		}
	}()

	var page *os.File
	if isFeederSource(options.Source) {
		args = append(args, "--batch="+filepath.Join(workingDir, "page-%04d.png"))
	} else {
		pagePath := filepath.Join(workingDir, "page-0001.png")
		page, err = os.Create(pagePath)
		if err != nil {
			return fmt.Errorf("create scanned page: %w", err)
		}
	}

	command := exec.CommandContext(ctx, "scanimage", args...)
	if page != nil {
		command.Stdout = page
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	runErr := command.Run()
	var closeErr error
	if page != nil {
		closeErr = page.Close()
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

	pagePaths, err := filepath.Glob(filepath.Join(workingDir, "page-*.png"))
	if err != nil {
		return fmt.Errorf("find scanned pages: %w", err)
	}
	if len(pagePaths) == 0 {
		return fmt.Errorf("scanimage produced no pages")
	}
	for _, pagePath := range pagePaths {
		info, err := os.Stat(pagePath)
		if err != nil {
			return fmt.Errorf("inspect scanned page: %w", err)
		}
		if info.Size() == 0 {
			return fmt.Errorf("scanimage produced an empty page")
		}
	}

	s.workingDir = workingDir
	cleanup = false
	return nil
}

func isFeederSource(source Source) bool {
	value := strings.ToLower(string(source))
	return strings.Contains(value, "adf") || strings.Contains(value, "feeder")
}

func (s *LinuxScanner) Save(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.workingDir == "" {
		return fmt.Errorf("no scanned page to save")
	}
	if path == "" {
		return fmt.Errorf("output path must not be empty")
	}

	pagePaths, err := filepath.Glob(filepath.Join(s.workingDir, "page-*.png"))
	if err != nil {
		return fmt.Errorf("find scanned pages: %w", err)
	}
	if len(pagePaths) == 0 {
		return fmt.Errorf("no scanned pages to save")
	}
	sort.Strings(pagePaths)

	temporaryDir, err := os.MkdirTemp(filepath.Dir(path), ".collate-save-")
	if err != nil {
		return fmt.Errorf("create temporary output directory: %w", err)
	}
	defer os.RemoveAll(temporaryDir)

	temporaryPath := filepath.Join(temporaryDir, filepath.Base(path))
	if err := pdf.ImportImages(pagePaths, temporaryPath); err != nil {
		return fmt.Errorf("create scanned PDF: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("save scanned PDF: %w", err)
	}

	return nil
}

func (s *LinuxScanner) Close(ctx context.Context) error {
	// TODO: implement close logic
	return nil
}

func getScannerCapabilities(ctx context.Context, deviceID string) (Capabilities, error) {
	capabilities := Capabilities{}
	output, err := exec.CommandContext(ctx, "scanimage", "-d", deviceID, "-A").Output()
	if err != nil || ctx.Err() != nil {
		return capabilities, err
	}

	optionPattern := regexp.MustCompile(`^\s*--([[:alnum:]-]+)\s+(.+)$`)
	numberPattern := regexp.MustCompile(`\d+(?:\.\d+)?`)

	sources := map[Source]bool{}
	var sourceOrder []Source
	modes := map[Mode]bool{}
	resolutions := map[int]bool{}
	var widthMM, heightMM float64

	for line := range strings.SplitSeq(string(output), "\n") {
		matches := optionPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		option, value := matches[1], strings.TrimSpace(matches[2])
		switch option {
		case "source":
			choices := strings.SplitN(value, "[", 2)[0]
			for choice := range strings.SplitSeq(choices, "|") {
				source := Source(strings.TrimSpace(choice))
				if source == "" || strings.HasPrefix(string(source), "<") || sources[source] {
					continue
				}
				sources[source] = true
				sourceOrder = append(sourceOrder, source)
			}
		case "mode":
			value = strings.ToLower(value)
			if strings.Contains(value, "color") {
				modes[ModeColor] = true
			}
			if strings.Contains(value, "gray") || strings.Contains(value, "grey") {
				modes[ModeGray] = true
			}
		case "resolution":
			for _, number := range numberPattern.FindAllString(value, -1) {
				resolution, err := strconv.ParseFloat(number, 64)
				if err == nil && resolution > 0 {
					resolutions[int(resolution)] = true
				}
			}
		case "page-width", "br-x":
			for _, number := range numberPattern.FindAllString(value, -1) {
				width, err := strconv.ParseFloat(number, 64)
				if err == nil && width > widthMM {
					widthMM = width
				}
			}
		case "page-height", "br-y":
			for _, number := range numberPattern.FindAllString(value, -1) {
				height, err := strconv.ParseFloat(number, 64)
				if err == nil && height > heightMM {
					heightMM = height
				}
			}
		}
	}

	capabilities.Sources = sourceOrder
	for _, mode := range []Mode{ModeColor, ModeGray} {
		if modes[mode] {
			capabilities.Modes = append(capabilities.Modes, mode)
		}
	}
	for _, paper := range []struct {
		paper  Paper
		width  float64
		height float64
	}{
		{PaperA4, 210, 297},
		{PaperA5, 148, 210},
		{PaperLetter, 215.9, 279.4},
	} {
		if widthMM >= paper.width && heightMM >= paper.height {
			capabilities.Papers = append(capabilities.Papers, paper.paper)
		}
	}
	for resolution := range resolutions {
		capabilities.Resolutions = append(capabilities.Resolutions, resolution)
	}
	sort.Ints(capabilities.Resolutions)

	return capabilities, nil
}
