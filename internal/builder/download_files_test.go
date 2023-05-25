package builder

import (
	"strings"
	"testing"
)

func TestDownloadFilesOperation_SendFiles(t *testing.T) {
	t.Run("it should emit each file downloaded with the Client", func(t *testing.T) {
		filesOnServer := map[string]string{
			"index.php": "index.php contents",
		}

		mockedClient := newMockedClient(filesOnServer)

		op, err := NewDownloadFilesOperation(mockedClient, "/srv/")
		if err != nil {
			t.Errorf("got error %v; want nil", err)
		}

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

	t.Run("it should return an error if the public path does not end in a forward slash", func(t *testing.T) {
		_, err := NewDownloadFilesOperation(newMockedClient(map[string]string{}), "/srv")
		if err == nil {
			t.Errorf("got nil error; want non-nil")
		}
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
