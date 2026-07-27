package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/slobbe/collate/internal/cli/components"
	"github.com/slobbe/collate/internal/collate"
	"github.com/slobbe/collate/internal/utils"
)

func selectScanner(ctx context.Context, input *bufio.Reader, output io.Writer, infos []collate.ScannerInfo) (collate.ScannerInfo, error) {
	if len(infos) == 0 {
		return collate.ScannerInfo{}, fmt.Errorf("no scanners found")
	}
	if len(infos) == 1 {
		fmt.Fprintf(output, "Using scanner: %s\n", scannerName(infos[0]))
		return infos[0], nil
	}

	options := make([]clicomponent.SelectOption[collate.ScannerInfo], len(infos))
	for index, info := range infos {
		options[index] = clicomponent.SelectOption[collate.ScannerInfo]{Value: info, Label: scannerName(info)}
	}
	return (clicomponent.Select[collate.ScannerInfo]{Prompt: "Select scanner", Options: options}).Run(ctx, input, output)
}

func selectSource(ctx context.Context, input *bufio.Reader, output io.Writer, sources []collate.ScanSource) (collate.ScanSource, error) {
	if len(sources) == 0 {
		return collate.ScanSource{}, fmt.Errorf("scanner does not advertise scan sources")
	}
	options := make([]clicomponent.SelectOption[collate.ScanSource], len(sources))
	for index, source := range sources {
		options[index] = clicomponent.SelectOption[collate.ScanSource]{Value: source, Label: sourceLabel(source)}
	}
	return (clicomponent.Select[collate.ScanSource]{Prompt: "Select source", Options: options}).Run(ctx, input, output)
}

func promptPaper(ctx context.Context, input *bufio.Reader, output io.Writer) (collate.Paper, error) {
	return (clicomponent.Select[collate.Paper]{
		Prompt: "Select paper",
		Options: []clicomponent.SelectOption[collate.Paper]{
			{Value: collate.PaperA4, Label: collate.PaperA4.Name},
			{Value: collate.PaperA5, Label: collate.PaperA5.Name},
			{Value: collate.PaperLetter, Label: collate.PaperLetter.Name},
		},
	}).Run(ctx, input, output)
}

func selectMode(ctx context.Context, input *bufio.Reader, output io.Writer, modes []string) (string, error) {
	if len(modes) == 0 {
		return "", fmt.Errorf("scanner does not advertise color modes")
	}
	options := make([]clicomponent.SelectOption[string], len(modes))
	for index, mode := range modes {
		options[index] = clicomponent.SelectOption[string]{Value: mode, Label: mode}
	}
	return (clicomponent.Select[string]{Prompt: "Select color mode", Options: options}).Run(ctx, input, output)
}

func selectResolution(ctx context.Context, input *bufio.Reader, output io.Writer, resolutions []int) (int, error) {
	if len(resolutions) == 0 {
		return 0, fmt.Errorf("scanner does not advertise scan resolutions")
	}
	options := make([]clicomponent.SelectOption[int], len(resolutions))
	for index, resolution := range resolutions {
		options[index] = clicomponent.SelectOption[int]{Value: resolution, Label: fmt.Sprintf("%d DPI", resolution)}
	}
	return (clicomponent.Select[int]{Prompt: "Select resolution", Options: options}).Run(ctx, input, output)
}

func promptOutputPath(ctx context.Context, input *bufio.Reader, output io.Writer, now time.Time) (string, error) {
	return (clicomponent.TextInput{
		Prompt:    "Output path",
		Default:   now.Format("scan_20060102_150405.pdf"),
		Normalize: utils.NormalizePDFPath,
	}).Run(ctx, input, output)
}

func promptBackOrder(ctx context.Context, input *bufio.Reader, output io.Writer) (collate.BackOrder, error) {
	return (clicomponent.Select[collate.BackOrder]{
		Prompt: "Select back-page order",
		Options: []clicomponent.SelectOption[collate.BackOrder]{
			{Value: collate.BackOrderReverse, Label: "Reverse"},
			{Value: collate.BackOrderForward, Label: "Forward"},
		},
	}).Run(ctx, input, output)
}

func scannerName(info collate.ScannerInfo) string {
	if info.Name != "" {
		return info.Name
	}
	return info.ID
}

func sourceLabel(source collate.ScanSource) string {
	if source.Name == "" {
		return source.ID
	}
	if source.ID == "" || strings.EqualFold(source.Name, source.ID) {
		return source.Name
	}
	return fmt.Sprintf("%s (%s)", source.Name, source.ID)
}
