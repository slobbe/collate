package pdf

import (
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	core "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Open loads path as a PDF document.
func Open(path string) (Document, error) {
	ctx, err := api.ReadContextFile(path)
	if err != nil {
		return nil, err
	}

	return &document{ctx: ctx}, nil
}

// New creates an empty PDF document.
func New() (Document, error) {
	conf := model.NewDefaultConfiguration()
	conf.Cmd = model.MERGECREATE
	conf.ValidationMode = model.ValidationRelaxed

	ctx, err := core.CreateContextWithXRefTable(conf, types.PaperSize["A4"])
	if err != nil {
		return nil, err
	}

	return &document{ctx: ctx}, nil
}

type document struct {
	ctx *model.Context
}

func (d *document) PageCount() int {
	return d.ctx.PageCount
}

func (d *document) PageAt(index int) (Page, error) {
	if index < 0 || index >= d.PageCount() {
		return nil, fmt.Errorf("page index %d out of range [0, %d)", index, d.PageCount())
	}

	return page{document: d, number: index + 1}, nil
}

func (d *document) AppendPage(source Page) error {
	page, ok := source.(page)
	if !ok {
		return fmt.Errorf("unsupported page type %T", source)
	}

	return core.AddPages(page.document.ctx, d.ctx, []int{page.number}, false)
}

func (d *document) Save(path string) error {
	return api.CreatePDFFile(d.ctx.XRefTable, path, d.ctx.Configuration)
}

type page struct {
	document *document
	number   int
}

func (page) PDFPage() {}

var _ Document = (*document)(nil)
var _ Page = page{}
