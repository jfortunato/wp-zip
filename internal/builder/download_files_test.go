package builder

import (
	"reflect"
	"strings"
	"testing"
)

func TestDownloadFilesOperation_SendFiles(t *testing.T) {
	t.Run("it should emit each file downloaded", func(t *testing.T) {
		filesOnServer := map[string]string{
			"index.php": "index.php contents",
		}

		mockedDownloader := &MockDirectoryDownloader{files: filesOnServer}

		operation, err := NewDownloadFilesOperation(mockedDownloader, "/srv/")
		if err != nil {
			t.Errorf("got error %v; want nil", err)
		}

		var filesSent []File

		operation.SendFiles(func(file File) error {
			filesSent = append(filesSent, file)
			return nil
		})

		// Assert the channel emitted the expected files
		expectFiles(t, filesSent, []File{
			{Name: "index.php", Body: strings.NewReader("index.php contents")},
		})
	})
}

func TestDownloadFiles_DirectoryDownloader_Factory(t *testing.T) {
	t.Run("it should return the proper DirectoryDownloader depending on the client's tar support", func(t *testing.T) {
		var tests = []struct {
			isTarSupported bool
			expectedType   string
		}{
			{isTarSupported: true, expectedType: "*builder.TarDirectoryDownloader"},
			{isTarSupported: false, expectedType: "*builder.SftpDirectoryDownloader"},
		}

		for _, test := range tests {
			downloader := NewDirectoryDownloader(&MockTarChecker{isTarSupported: test.isTarSupported})

			if reflect.TypeOf(downloader).String() != test.expectedType {
				t.Errorf("got type %s; want %s", reflect.TypeOf(downloader).String(), test.expectedType)
			}
		}
	})
}

type MockTarChecker struct {
	isTarSupported bool
}

func (m *MockTarChecker) HasTar() bool {
	return m.isTarSupported
}

type MockDirectoryDownloader struct {
	files map[string]string
}

func (m *MockDirectoryDownloader) Download(src string, fn DownloadFunc) error {
	for name, body := range m.files {
		fn(name, strings.NewReader(body))
	}
	return nil
}

func expectFiles(t *testing.T, files []File, expectedFiles []File) {
	if len(files) != len(expectedFiles) {
		t.Errorf("got %d files; want %d", len(files), len(expectedFiles))
	}

	for i, file := range files {
		if file.Name != expectedFiles[i].Name {
			t.Errorf("got file %d name %s; want %s", i, file.Name, expectedFiles[i].Name)
		}

		var expectedBody string
		var actualBody string
		expectedFiles[i].Body.Read([]byte(expectedBody))
		file.Body.Read([]byte(actualBody))
		if actualBody != expectedBody {
			t.Errorf("got file %d body %s; want %s", i, actualBody, expectedBody)
		}
	}
}
