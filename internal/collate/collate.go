package collate

import (
	"fmt"

	"github.com/slobbe/collate/internal/pdf/pdfcpu"
)

type BackOrder string

const (
	BackOrderReverse BackOrder = "reverse"
	BackOrderForward BackOrder = "forward"
)

func ParseBackOrder(value string) (BackOrder, error) {
	order := BackOrder(value)
	switch order {
	case BackOrderReverse, BackOrderForward:
		return order, nil
	default:
		return "", fmt.Errorf("invalid back order %q (must be %q or %q)", value, BackOrderReverse, BackOrderForward)
	}
}

func Collate(
	frontPath string,
	backPath string,
	outputPath string,
	backOrder BackOrder,
) error {
	if _, err := ParseBackOrder(string(backOrder)); err != nil {
		return err
	}

	pdf := pdfcpu.Engine{}

	front, err := pdf.Open(frontPath)
	if err != nil {
		return fmt.Errorf("open front PDF: %w", err)
	}

	back, err := pdf.Open(backPath)
	if err != nil {
		return fmt.Errorf("open back PDF: %w", err)
	}

	if back.PageCount() > front.PageCount() || front.PageCount() > back.PageCount()+1 {
		return fmt.Errorf(
			"page count mismatch: front has %d pages, back has %d; expected equal counts or one extra front page",
			front.PageCount(),
			back.PageCount(),
		)
	}

	output, err := pdf.New()
	if err != nil {
		return fmt.Errorf("create output PDF: %w", err)
	}

	for i := 0; i < front.PageCount(); i++ {
		frontPage, err := front.PageAt(i)
		if err != nil {
			return fmt.Errorf("read front page %d: %w", i+1, err)
		}
		if err := output.AppendPage(frontPage); err != nil {
			return fmt.Errorf("add front page %d: %w", i+1, err)
		}

		if i >= back.PageCount() {
			continue
		}

		backIndex := i
		if backOrder == BackOrderReverse {
			backIndex = back.PageCount() - 1 - i
		}

		backPage, err := back.PageAt(backIndex)
		if err != nil {
			return fmt.Errorf("read back page %d: %w", backIndex+1, err)
		}
		if err := output.AppendPage(backPage); err != nil {
			return fmt.Errorf("add back page %d: %w", backIndex+1, err)
		}
	}

	if err := output.Save(outputPath); err != nil {
		return fmt.Errorf("save output PDF: %w", err)
	}

	return nil
}
