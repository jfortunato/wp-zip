package builder

import (
	"strings"
	"testing"
)

func TestDownloadFilesOperation_SendFiles(t *testing.T) {
	t.Run("it should emit each file downloaded with the Client", func(t *testing.T) {
		mockedClient := &MockClient{
			DownloadFunc: func(path string, ch chan File) error {
				ch <- File{Name: "index.php", Body: strings.NewReader("index.php contents")}
				close(ch)
				return nil
			},
		}
		op := NewDownloadFilesOperation(mockedClient, "/srv")

		ch, err := op.SendFiles()
		if err != nil {
			t.Errorf("got error %v; want nil", err)
		}

		// Transform the channel into a slice of files
		var files []File
		for file := range ch {
			files = append(files, file)
		}

		// Assert the channel emitted the expected files
		expectFiles(t, files, []File{
			{Name: "index.php", Body: strings.NewReader("index.php contents")},
		})
	})

	t.Run("it should automatically add a forward slash to the public path", func(t *testing.T) {
	})
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
