package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/slobbe/collate/internal/cli/utils"
	"github.com/slobbe/collate/internal/collate"
	"github.com/slobbe/collate/internal/scanner"
	"github.com/slobbe/collate/internal/utils"
)

func selectScanner(ctx context.Context, input *bufio.Reader, output io.Writer, infos []scanner.Info) (scanner.Info, error) {
	if len(infos) == 0 {
		return scanner.Info{}, fmt.Errorf("no scanners found")
	}
	if len(infos) == 1 {
		fmt.Fprintf(output, "Using scanner: %s\n", scannerName(infos[0]))
		return infos[0], nil
	}

	options := make([]cliutils.SelectOption[scanner.Info], len(infos))
	for index, info := range infos {
		options[index] = cliutils.SelectOption[scanner.Info]{Value: info, Label: scannerName(info)}
	}
	return (cliutils.Select[scanner.Info]{Prompt: "Select scanner", Options: options}).Run(ctx, input, output)
}

func selectSource(ctx context.Context, input *bufio.Reader, output io.Writer, sources []scanner.Source) (scanner.Source, error) {
	if len(sources) == 0 {
		return scanner.Source{}, fmt.Errorf("scanner does not advertise scan sources")
	}
	options := make([]cliutils.SelectOption[scanner.Source], len(sources))
	for index, source := range sources {
		options[index] = cliutils.SelectOption[scanner.Source]{Value: source, Label: sourceLabel(source)}
	}
	return (cliutils.Select[scanner.Source]{Prompt: "Select source", Options: options}).Run(ctx, input, output)
}

func promptPaper(ctx context.Context, input *bufio.Reader, output io.Writer) (scanner.Paper, error) {
	return (cliutils.Select[scanner.Paper]{
		Prompt: "Select paper",
		Options: []cliutils.SelectOption[scanner.Paper]{
			{Value: scanner.PaperA4, Label: scanner.PaperA4.Name},
			{Value: scanner.PaperA5, Label: scanner.PaperA5.Name},
			{Value: scanner.PaperLetter, Label: scanner.PaperLetter.Name},
		},
	}).Run(ctx, input, output)
}

func selectMode(ctx context.Context, input *bufio.Reader, output io.Writer, modes []string) (string, error) {
	if len(modes) == 0 {
		return "", fmt.Errorf("scanner does not advertise color modes")
	}
	options := make([]cliutils.SelectOption[string], len(modes))
	for index, mode := range modes {
		options[index] = cliutils.SelectOption[string]{Value: mode, Label: mode}
	}
	return (cliutils.Select[string]{Prompt: "Select color mode", Options: options}).Run(ctx, input, output)
}

func selectResolution(ctx context.Context, input *bufio.Reader, output io.Writer, resolutions []int) (int, error) {
	if len(resolutions) == 0 {
		return 0, fmt.Errorf("scanner does not advertise scan resolutions")
	}
	options := make([]cliutils.SelectOption[int], len(resolutions))
	for index, resolution := range resolutions {
		options[index] = cliutils.SelectOption[int]{Value: resolution, Label: fmt.Sprintf("%d DPI", resolution)}
	}
	return (cliutils.Select[int]{Prompt: "Select resolution", Options: options}).Run(ctx, input, output)
}

func promptOutputPath(ctx context.Context, input *bufio.Reader, output io.Writer, now time.Time) (string, error) {
	return (cliutils.TextInput{
		Prompt:    "Output path",
		Default:   now.Format("scan_20060102_150405.pdf"),
		Normalize: utils.NormalizePDFPath,
	}).Run(ctx, input, output)
}

func promptBackOrder(ctx context.Context, input *bufio.Reader, output io.Writer) (collate.BackOrder, error) {
	return (cliutils.Select[collate.BackOrder]{
		Prompt: "Select back-page order",
		Options: []cliutils.SelectOption[collate.BackOrder]{
			{Value: collate.BackOrderReverse, Label: "Reverse"},
			{Value: collate.BackOrderForward, Label: "Forward"},
		},
	}).Run(ctx, input, output)
}

func scannerName(info scanner.Info) string {
	if info.Name != "" {
		return info.Name
	}
	return info.ID
}

func sourceLabel(source scanner.Source) string {
	if source.Name == "" {
		return source.ID
	}
	if source.ID == "" || strings.EqualFold(source.Name, source.ID) {
		return source.Name
	}
	return fmt.Sprintf("%s (%s)", source.Name, source.ID)
}
