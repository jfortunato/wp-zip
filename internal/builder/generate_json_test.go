package builder

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestGenerateJsonOperation(t *testing.T) {
	t.Run("it sends a json file containing metadata about the sites environment", func(t *testing.T) {
		operation := newOperation()

		expectFilesSentFromOperation(t, operation, map[string]string{
			"wpmigrate-export.json": `{"name":"Migrated Site"}`,
		})
	})

	t.Run("it returns an error if we cannot upload the php file", func(t *testing.T) {
		operation := newOperation()
		operation.u = &MockFileUploadDeleter{uploadErrorStub: errors.New("error upload")}

		err := operation.SendFiles(func(file File) error {
			return nil
		})
		assertError(t, err, ErrCouldNotUploadFile)
	})

	t.Run("it returns an error if the http request returns an error", func(t *testing.T) {
		operation := newOperation()
		operation.g = &MockHttpGetter{
			responseStubs: map[string]GetterResponse{
				"https://localhost/wp-zip.php": {
					resp: nil,
					err:  errors.New("error response"),
				},
			},
		}

		err := operation.SendFiles(func(file File) error {
			return nil
		})

		assertError(t, err, ErrInvalidResponse)
	})

	t.Run("it attempts an insecure url (http) after an unsuccessful secure url (https)", func(t *testing.T) {
		operation := newOperation()
		operation.g = &MockHttpGetter{
			responseStubs: map[string]GetterResponse{
				"https://localhost/wp-zip.php": {
					resp: nil,
					err:  errors.New("error response"),
				},
				"http://localhost/wp-zip.php": {
					resp: io.NopCloser(strings.NewReader(`{"name":"Migrated Site"}`)),
					err:  nil,
				},
			},
		}

		expectFilesSentFromOperation(t, operation, map[string]string{
			"wpmigrate-export.json": `{"name":"Migrated Site"}`,
		})
	})

	t.Run("it returns an error if the http response is not what we expect", func(t *testing.T) {
		operation := newOperation()
		// No error, but the response is not what we expect
		operation.g = &MockHttpGetter{
			responseStubs: map[string]GetterResponse{
				"https://localhost/wp-zip.php": {
					resp: io.NopCloser(strings.NewReader("invalid response")),
					err:  nil,
				},
			},
		}

		err := operation.SendFiles(func(file File) error {
			return nil
		})

		assertError(t, err, ErrUnexpectedResponse)
	})

	t.Run("it returns an error if the uploaded file cannot be deleted", func(t *testing.T) {
		operation := newOperation()
		operation.u = &MockFileUploadDeleter{deleteErrorStub: errors.New("error delete")}

		err := operation.SendFiles(func(file File) error {
			return nil
		})

		assertError(t, err, ErrCouldNotDeleteFile)
	})
}

// By default, we will create a completely valid operation. The client code can then override
// the default behaviour by setting the fields on the operation.
func newOperation() *GenerateJsonOperation {
	return &GenerateJsonOperation{
		u: &MockFileUploadDeleter{},
		g: &MockHttpGetter{
			responseStubs: map[string]GetterResponse{
				"https://localhost/wp-zip.php": {
					resp: io.NopCloser(strings.NewReader(`{"name":"Migrated Site"}`)),
					err:  nil,
				},
			},
		},
		publicUrl:  "localhost",
		publicPath: "public",
		randomFileNamer: func() string {
			return "wp-zip.php"
		},
	}
}

type MockFileUploadDeleter struct {
	uploadErrorStub error
	deleteErrorStub error
}

func (m *MockFileUploadDeleter) Upload(r io.Reader, dst string) error {
	return m.uploadErrorStub
}

func (m *MockFileUploadDeleter) Delete(dst string) error {
	return m.deleteErrorStub
}

type GetterResponse struct {
	resp io.ReadCloser
	err  error
}

type MockHttpGetter struct {
	responseStubs map[string]GetterResponse
}

func (m *MockHttpGetter) Get(url string) (resp io.ReadCloser, err error) {
	response, ok := m.responseStubs[url]
	if !ok {
		return nil, errors.New("no response stub for url: " + url)
	}
	return response.resp, response.err
}

func assertError(t testing.TB, got, want error) {
	t.Helper()
	if got == nil {
		t.Fatal("didn't get an error but wanted one")
	}

	if !errors.Is(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
