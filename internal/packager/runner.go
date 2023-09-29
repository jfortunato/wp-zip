package packager

import (
	"archive/zip"
	"errors"
	"fmt"
	"github.com/jfortunato/wp-zip/internal/operations"
	"io"
)

var ErrNoOperations = errors.New("no operations to run")

// Runner is responsible for running the operations that will create the zip archive. It runs the operations sequentially, and writes the files that they send
// one by one into the zip archive.
type Runner struct{}

func (r *Runner) Run(ops []operations.Operation, writer io.Writer) error {
	if len(ops) == 0 {
		return ErrNoOperations
	}

	// Create a new zip writer
	zw := zip.NewWriter(writer)
	defer zw.Close()

	for _, operation := range ops {
		err := operation.SendFiles(func(file operations.File) error {
			// Write the files into the zip
			err := writeIntoZip(zw, file)
			if err != nil {
				return fmt.Errorf("error writing file %s into zip: %s", file.Name, err)
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("error sending files: %s", err)
		}
	}

	return nil
}

func writeIntoZip(zw *zip.Writer, file operations.File) error {
	f, err := zw.Create(file.Name)
	if err != nil {
		return fmt.Errorf("error creating file %s in zip: %s", file.Name, err)
	}
	_, err = io.Copy(f, file.Body)
	if err != nil {
		return fmt.Errorf("error copying file %s into zip: %s", file.Name, err)
	}
	return nil
}
