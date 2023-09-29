package operations

import (
	"io"
)

type SendFilesFunc func(file File) error

// Operation represents a single operation that the builder can run. For example, exporting the database, or downloading the site files.
type Operation interface {
	SendFiles(fn SendFilesFunc) error
}

type File struct {
	Name string
	Body io.Reader
}
