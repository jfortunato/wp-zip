package builder

import (
	"archive/zip"
	"fmt"
	"github.com/jfortunato/wp-zip/internal/operations"
	"io"
)

type ErrNoOperations struct{}

func (e *ErrNoOperations) Error() string {
	return "no operations to run"
}

func BuildZip(writer io.Writer, operationsToRun []operations.Operation) error {
	// Ensure we have at least one operation to run
	if len(operationsToRun) == 0 {
		return &ErrNoOperations{}
	}

	// Create a new zip writer
	zw := zip.NewWriter(writer)
	defer zw.Close()

	// Run each operation
	for _, operation := range operationsToRun {
		err := operation.WriteIntoZip(zw)
		if err != nil {
			return fmt.Errorf("error running operation: %s", err)
		}
	}

	return nil
}

func WriteIntoZip(zw *zip.Writer, filename string, contents io.Reader) error {
	f, err := zw.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file %s in zip: %s", filename, err)
	}
	_, err = io.Copy(f, contents)
	if err != nil {
		return fmt.Errorf("error copying file %s into zip: %s", filename, err)
	}
	return nil
}
