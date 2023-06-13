package builder

import (
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDownloadFilesOperation_SendFiles(t *testing.T) {
	t.Run("it should emit each file downloaded", func(t *testing.T) {
		filesOnServer := map[string]string{
			"index.php": "index.php contents",
		}

		mockedEmitter := &MockFileEmitter{files: filesOnServer}

		operation, err := NewDownloadFilesOperation(mockedEmitter, "/srv/")
		if err != nil {
			t.Errorf("got error %v; want nil", err)
		}

		expectFilesSentFromOperation(t, operation, map[string]string{
			"files/index.php": "index.php contents",
		})
	})
}

func TestDownloadFiles_FileEmitter_Factory(t *testing.T) {
	t.Run("it should return the proper FileEmitter depending on the client's tar support", func(t *testing.T) {
		var tests = []struct {
			isTarSupported bool
			expectedType   string
		}{
			{isTarSupported: true, expectedType: "*builder.TarFileEmitter"},
			{isTarSupported: false, expectedType: "*builder.SftpFileEmitter"},
		}

		for _, test := range tests {
			downloader := NewFileEmitter(&MockTarChecker{isTarSupported: test.isTarSupported}, &ClientStub{})

			if reflect.TypeOf(downloader).String() != test.expectedType {
				t.Errorf("got type %s; want %s", reflect.TypeOf(downloader).String(), test.expectedType)
			}
		}
	})
}

type ClientStub struct {
}

func (c *ClientStub) ReadDir(path string) ([]os.FileInfo, error) {
	return nil, nil
}

func (c *ClientStub) Open(path string) (*sftp.File, error) {
	return nil, nil
}

func (c *ClientStub) NewSession() (*ssh.Session, error) {
	return nil, nil
}

func TestTarFileEmitter_EmitAll(t *testing.T) {
	t.Run("it should emit the directory using tar and untar/convert into files", func(t *testing.T) {
	})
}

func TestSftpFileEmitter_EmitAll(t *testing.T) {
	t.Run("it should emit the directory using sftp", func(t *testing.T) {
	})
}

type MockTarChecker struct {
	isTarSupported bool
}

func (m *MockTarChecker) HasTar() bool {
	return m.isTarSupported
}

type MockFileEmitter struct {
	files map[string]string
}

func (m *MockFileEmitter) EmitAll(src string, fn EmitFunc) error {
	for name, body := range m.files {
		fn(name, strings.NewReader(body))
	}
	return nil
}

func (m *MockFileEmitter) EmitSingle(src string, fn EmitFunc) error {
	basename := filepath.Base(src)
	// If the file doesn't exist, return an error
	if _, ok := m.files[basename]; !ok {
		return os.ErrNotExist
	}

	return nil
}

func expectFilesSentFromOperation(t *testing.T, operation Operation, expectedFiles map[string]string) {
	var filesSent []File

	operation.SendFiles(func(file File) error {
		filesSent = append(filesSent, file)
		return nil
	})

	// Convert the expected files to a File slice
	var expectedFilesSlice []File
	for name, body := range expectedFiles {
		expectedFilesSlice = append(expectedFilesSlice, File{Name: name, Body: strings.NewReader(body)})
	}

	expectFiles(t, filesSent, expectedFilesSlice)
}

func expectFiles(t *testing.T, files []File, expectedFiles []File) {
	if len(files) != len(expectedFiles) {
		t.Errorf("got %d files; want %d", len(files), len(expectedFiles))
	}

	for i, file := range files {
		if file.Name != expectedFiles[i].Name {
			t.Errorf("got file %d name %s; want %s", i, file.Name, expectedFiles[i].Name)
		}

		expectedBody := readerToString(expectedFiles[i].Body)
		actualBody := readerToString(file.Body)

		if actualBody != expectedBody {
			t.Errorf("got file %d body %s; want %s", i, actualBody, expectedBody)
		}
	}
}
