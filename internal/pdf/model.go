package pdf

// Page is an opaque reference to a page in a source document.
type Page interface {
	PDFPage()
}

// Document is an in-memory PDF document.
type Document interface {
	PageCount() int
	PageAt(index int) (Page, error)
	AppendPage(page Page) error
	Save(path string) error
}

type Engine interface {
	Open(path string) (Document, error)
	New() (Document, error)
}
