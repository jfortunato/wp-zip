package builder

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageWP(t *testing.T) {
	t.Run("it runs the appropriate operations to build a zip archive", func(t *testing.T) {
		filesOnServer := map[string]string{
			"index.php":     "index.php contents",
			"wp-config.php": "wp-config.php contents",
		}

		mockedClient := newMockedClient(filesOnServer)
		operations := []Operation{&MockOperation{filesToSend: filesOnServer}}

		b := &bytes.Buffer{}

		PackageWP(mockedClient, b, "/var/www/html/", operations)

		expectZipContents(t, b, prefixFiles("files/", filesOnServer))
	})

	t.Run("it should not attempt any operations if it cannot read the wp-config file", func(t *testing.T) {
		filesOnServer := map[string]string{
			"index.php": "index.php contents",
		}

		mockedClient := newMockedClient(filesOnServer)
		operation := &MockOperation{}

		b := &bytes.Buffer{}

		PackageWP(mockedClient, b, "/var/www/html/", []Operation{operation})

		// Assert that the operation was not called
		if operation.sendFilesCalled != 0 {
			t.Errorf("operation.SendFiles() was called %d times; want 0", operation.sendFilesCalled)
		}

		// Assert the buffer is empty
		if b.Len() != 0 {
			t.Errorf("buffer is not empty; want empty buffer")
		}
	})

	t.Run("empty operations", func(t *testing.T) {
		mockedClient := newMockedClient(nil)

		b := &bytes.Buffer{}

		err := PackageWP(mockedClient, b, "/var/www/html/", []Operation{})

		_, ok := err.(*ErrNoOperations)
		if !ok {
			t.Errorf("got error %v; want ErrNoOperations", err)
		}
	})

	t.Run("multiple operations", func(t *testing.T) {
		filesOnServer := map[string]string{
			"index.php":     "index.php contents",
			"wp-config.php": "wp-config.php contents",
		}

		mockedClient := newMockedClient(filesOnServer)
		operations := []Operation{
			&MockOperation{filesToSend: map[string]string{"index.php": "index.php contents"}},
			&MockOperation{filesToSend: map[string]string{"wp-config.php": "wp-config.php contents"}},
		}

		b := &bytes.Buffer{}

		PackageWP(mockedClient, b, "/var/www/html/", operations)

		expectZipContents(t, b, prefixFiles("files/", filesOnServer))
	})
}

func TestPublicPath_String(t *testing.T) {
	t.Run("it should always end in forward slash when converting to string", func(t *testing.T) {
		var tests = []struct {
			input    string
			expected string
		}{
			{"/srv/", "/srv/"},
			{"/srv", "/srv/"},
			{"/", "/"},
		}

		for _, test := range tests {
			path := PublicPath(test.input)
			if path.String() != test.expected {
				t.Errorf("got %s; want %s", path.String(), test.expected)
			}
		}
	})
}

type MockClient struct {
	UploadFunc   func(src io.Reader, dst string) error
	DownloadFunc func(src string, ch chan File) error
	RunFunc      func(cmd string) ([]byte, error)
	filesystem   map[string]string
}

func (c *MockClient) Upload(src io.Reader, dst string) error {
	return c.UploadFunc(src, dst)
}

func (c *MockClient) Download(src string, ch chan File) error {
	return c.DownloadFunc(src, ch)
}

func (c *MockClient) Run(cmd string) ([]byte, error) {
	return c.RunFunc(cmd)
}

type MockOperation struct {
	filesToSend     map[string]string
	sendFilesCalled int
}

func (o *MockOperation) SendFiles(fn SendFilesFunc) error {
	o.sendFilesCalled++

	for filename, contents := range o.filesToSend {
		f := File{
			Name: filename,
			Body: strings.NewReader(contents),
		}

		if err := fn(f); err != nil {
			return err
		}
	}

	return nil
}

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

func prefixFiles(prefix string, files map[string]string) map[string]string {
	prefixedFiles := make(map[string]string)

	for filename, contents := range files {
		prefixedFiles[prefix+filename] = contents
	}

	return prefixedFiles
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
