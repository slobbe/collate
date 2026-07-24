// Package scanner discovers scanners available to collate.
package scanner

// Info describes a discovered scanner.
type Info struct {
	Device      string
	Description string
}

// Scanner is a scanner discovered on the current system.
type Scanner interface {
	Info() Info
}

type discoveredScanner struct {
	info Info
}

func (scanner discoveredScanner) Info() Info {
	return scanner.info
}
