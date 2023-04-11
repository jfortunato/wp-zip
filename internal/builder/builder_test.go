package builder

import (
	"archive/zip"
	"bytes"
	"github.com/jfortunato/wp-zip/internal/operations"
	"testing"
)

func TestBuildZip(t *testing.T) {
	t.Run("simple operation", func(t *testing.T) {
		files := map[string]string{
			"file1.txt": "file1 contents",
		}

		operationsToRun := []operations.Operation{
			&dummyOperation{files},
		}

		expectZipContents(t, operationsToRun, files)
	})

	t.Run("empty operations", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		operationsToRun := []operations.Operation{}

		err := BuildZip(buffer, operationsToRun)

		// Assert that the error is of type ErrNoOperations
		_, ok := err.(*ErrNoOperations)
		if !ok {
			t.Errorf("BuildZip() returned error of type %T; want ErrNoOperations", err)
		}
	})

	t.Run("multiple operations", func(t *testing.T) {
		operationsToRun := []operations.Operation{
			&dummyOperation{
				files: map[string]string{"file1.txt": "file1 contents"},
			},
			&dummyOperation{
				files: map[string]string{"file2.txt": "file2 contents"},
			},
		}

		expectZipContents(t, operationsToRun, map[string]string{
			"file1.txt": "file1 contents",
			"file2.txt": "file2 contents",
		})
	})

}

type dummyOperation struct {
	files map[string]string
}

func (o *dummyOperation) WriteIntoZip(zw *zip.Writer) error {
	for filename, contents := range o.files {
		WriteIntoZip(zw, filename, bytes.NewReader([]byte(contents)))
	}

	return nil
}

func expectZipContents(t *testing.T, operationsToRun []operations.Operation, contents map[string]string) {
	buffer := &bytes.Buffer{}
	err := BuildZip(buffer, operationsToRun)
	if err != nil {
		t.Errorf("BuildZip() returned error: %s", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Errorf("zip.NewReader() returned error: %s", err)
	}

	// Assert the zip file contains *exactly* the expected number of files
	expectedNumFiles := len(contents)
	if len(zr.File) != expectedNumFiles {
		t.Errorf("zip file contains %d files; want %d", len(zr.File), expectedNumFiles)
	}

	// Assert the zip file contains the expected files and their contents
	for filename, expectedContents := range contents {
		actualFile, err := zr.Open(filename)
		if err != nil {
			t.Errorf("zip file does not contain file %s", filename)
		}
		defer actualFile.Close()

		var b bytes.Buffer
		_, err = b.ReadFrom(actualFile)
		if err != nil {
			t.Errorf("error reading from file %s: %s", filename, err)
		}

		actualContents := b.String()
		if actualContents != expectedContents {
			t.Errorf("file %s contains %s; want %s", filename, actualContents, expectedContents)
		}
	}
}
