package packager

import (
	"archive/zip"
	"bytes"
	"errors"
	"github.com/jfortunato/wp-zip/internal/operations"
	"strings"
	"testing"
)

func TestRunner_Run(t *testing.T) {
	t.Run("it runs the operations to build a zip archive", func(t *testing.T) {
		filesOnServer := map[string]string{
			"index.php":     "index.php contents",
			"wp-config.php": "wp-config.php contents",
		}

		ops := []operations.Operation{&MockOperation{filesToSend: filesOnServer}}

		runner := &Runner{}

		b := &bytes.Buffer{}
		err := runner.Run(ops, b)

		if err != nil {
			t.Errorf("got error %v; want nil", err)
		}

		expectZipContents(t, b, filesOnServer)
	})

	t.Run("it should return an error if there are no operations", func(t *testing.T) {
		runner := &Runner{}

		err := runner.Run([]operations.Operation{}, nil)

		if !errors.Is(err, ErrNoOperations) {
			t.Errorf("got error %v; want ErrNoOperations", err)
		}
	})

	t.Run("it should return any error during an operations send files", func(t *testing.T) {
		// Use an ErrorOperation to force an error
		ops := []operations.Operation{&ErrorOperation{}}

		runner := &Runner{}

		b := &bytes.Buffer{}
		err := runner.Run(ops, b)

		if err == nil || !strings.Contains(err.Error(), "error from ErrorOperation") {
			t.Errorf("got error %v; want error from ErrorOperation", err)
		}

		expectZipContents(t, b, map[string]string{})
	})

	t.Run("multiple operations", func(t *testing.T) {
		filesOnServer := map[string]string{
			"index.php":     "index.php contents",
			"wp-config.php": "wp-config.php contents",
		}

		ops := []operations.Operation{
			&MockOperation{filesToSend: map[string]string{"index.php": "index.php contents"}},
			&MockOperation{filesToSend: map[string]string{"wp-config.php": "wp-config.php contents"}},
		}

		runner := &Runner{}

		b := &bytes.Buffer{}
		err := runner.Run(ops, b)

		if err != nil {
			t.Errorf("got error %v; want nil", err)
		}

		expectZipContents(t, b, filesOnServer)
	})
}

type MockOperation struct {
	filesToSend     map[string]string
	sendFilesCalled int
}

func (o *MockOperation) SendFiles(fn operations.SendFilesFunc) error {
	o.sendFilesCalled++

	for filename, contents := range o.filesToSend {
		f := operations.File{
			Name: filename,
			Body: strings.NewReader(contents),
		}

		if err := fn(f); err != nil {
			return err
		}
	}

	return nil
}

type ErrorOperation struct{}

func (o *ErrorOperation) SendFiles(fn operations.SendFilesFunc) error {
	return errors.New("error from ErrorOperation")
}

func expectZipContents(t *testing.T, b *bytes.Buffer, expectedFiles map[string]string) {
	zr, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err != nil {
		t.Errorf("zip.NewReader() returned error: %s", err)
	}

	// Assert the zip file contains *exactly* the expected number of files
	expectedNumFiles := len(expectedFiles)
	if len(zr.File) != expectedNumFiles {
		t.Errorf("zip file contains %d files; want %d", len(zr.File), expectedNumFiles)
	}

	// Assert the zip file contains the expected files and their contents
	for filename, expectedContents := range expectedFiles {
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
