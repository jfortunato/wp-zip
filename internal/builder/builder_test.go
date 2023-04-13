package builder

import (
	"archive/zip"
	"bytes"
	"errors"
	"github.com/jfortunato/wp-zip/internal/operations"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

type MockClient struct {
	UploadFunc    func(src io.Reader, dst string) error
	DownloadFunc  func(src string, ch chan File) error
	RunFunc       func(cmd string) ([]byte, error)
	filesystem    map[string]string
	downloadCalls int
}

func (c *MockClient) Upload(src io.Reader, dst string) error {
	return c.UploadFunc(src, dst)
}

func (c *MockClient) Download(src string, ch chan File) error {
	c.downloadCalls++
	return c.DownloadFunc(src, ch)
}

func (c *MockClient) Run(cmd string) ([]byte, error) {
	return c.RunFunc(cmd)
}

type MockOperation struct {
	SendFilesFunc func(ch chan File) error
}

func (o *MockOperation) SendFiles(ch chan File) error {
	return o.SendFilesFunc(ch)
}

func assertZipContents(t *testing.T, b *bytes.Buffer, expectedFiles map[string]string) {
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

//func setupSubTest() (fn) {
//	return func() {
//	}
//}

//func setupTest(t testing.T) func(t testing.T) {
//
//	// Return a function to teardown the test
//	return func(t testing.T) {
//		log.Println("teardown suite")
//		mockedClient.filesystem = nil
//		mockedClient.downloadCalls = 0
//	}
//}

func newMockedClient(filesystem map[string]string) *MockClient {
	mockedClient := new(MockClient)
	mockedClient.filesystem = filesystem
	mockedClient.DownloadFunc = func(src string, ch chan File) error {
		// If the src ends in a slash, send all files; otherwise, send only the file
		// that matches the src.
		if strings.HasSuffix(src, "/") {
			if len(mockedClient.filesystem) == 0 {
				return errors.New("no files found")
			}

			for filename, contents := range mockedClient.filesystem {
				ch <- File{
					Name: filename,
					Body: strings.NewReader(contents),
				}
			}
		} else {
			src = filepath.Base(src)

			if _, ok := mockedClient.filesystem[src]; !ok {
				return errors.New("file not found: " + src)
			}

			ch <- File{
				Name: src,
				Body: strings.NewReader(mockedClient.filesystem[src]),
			}
		}

		close(ch)
		return nil
	}

	return mockedClient
}

func newMockedOperation(filesToSend map[string]string) *MockOperation {
	mockedOperation := new(MockOperation)
	mockedOperation.SendFilesFunc = func(ch chan File) error {
		for filename, contents := range filesToSend {
			ch <- File{
				Name: filename,
				Body: strings.NewReader(contents),
			}
		}

		close(ch)
		return nil
	}

	return mockedOperation
}

func prefixFiles(prefix string, files map[string]string) map[string]string {
	prefixedFiles := make(map[string]string)

	for filename, contents := range files {
		prefixedFiles[prefix+filename] = contents
	}

	return prefixedFiles
}

func TestPackageWP(t *testing.T) {

	t.Run("it runs the appropriate operations to build a zip archive", func(t *testing.T) {
		filesOnServer := map[string]string{
			"index.php":     "index.php contents",
			"wp-config.php": "wp-config.php contents",
		}

		mockedClient := newMockedClient(filesOnServer)
		operations := []Operation{newMockedOperation(filesOnServer)}

		b := &bytes.Buffer{}

		PackageWP(mockedClient, b, "/var/www/html/", operations)

		assertZipContents(t, b, prefixFiles("files/", filesOnServer))
	})

	t.Run("it adds a trailing slash to the public path if not given", func(t *testing.T) {
		filesOnServer := map[string]string{
			"index.php":     "index.php contents",
			"wp-config.php": "wp-config.php contents",
		}

		mockedClient := newMockedClient(filesOnServer)
		operations := []Operation{newMockedOperation(filesOnServer)}

		b := &bytes.Buffer{}

		PackageWP(mockedClient, b, "/var/www/html", operations)

		assertZipContents(t, b, prefixFiles("files/", filesOnServer))
	})

	t.Run("it should not attempt any operations if it cannot read the wp-config file", func(t *testing.T) {
		filesOnServer := map[string]string{
			"index.php": "index.php contents",
		}

		mockedClient := newMockedClient(filesOnServer)
		operations := []Operation{&MockOperation{
			SendFilesFunc: func(ch chan File) error {
				panic("should not be called")
			},
		}}

		b := &bytes.Buffer{}

		PackageWP(mockedClient, b, "/var/www/html/", operations)

		// Assert only 1 call to Download() was made
		//if mockedClient.downloadCalls != 1 {
		//	t.Errorf("Download() was called %d times; want 1", mockedClient.downloadCalls)
		//}

		// Assert the buffer is empty
		if b.Len() != 0 {
			t.Errorf("buffer is not empty; want empty buffer")
		}
	})
}

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
