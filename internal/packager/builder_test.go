package packager

import (
	"errors"
	"github.com/jfortunato/wp-zip/internal/database"
	"github.com/jfortunato/wp-zip/internal/emitter"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"testing"
)

func TestBuilder_Build(t *testing.T) {
	t.Run("it should build the operations", func(t *testing.T) {
		builder := createBuilderWithStubs()

		ops, err := builder.Build(Options{})

		if err != nil {
			t.Errorf("got error %v; want nil", err)
		}

		// Assert that we receive a slice of operations, but don't create a potentially brittle test by asserting the exact operations
		if len(ops) == 0 {
			t.Errorf("got 0 operations; want > 0")
		}
	})

	t.Run("it should return an error if the credentials parser fails", func(t *testing.T) {
		builder := createBuilderWithStubs()
		builder.p = &CredentialsParserStub{errorStub: errors.New("error")}

		_, err := builder.Build(Options{})

		// Assert that we got the error we expect
		if !errors.Is(err, ErrCannotParseCredentials) {
			t.Errorf("got error %v; want ErrCannotParseCredentials", err)
		}
	})
}

func createBuilderWithStubs() *Builder {
	return &Builder{
		c: &ClientStub{},
		e: &FileEmitterStub{},
		g: &HttpGetterStub{},
		p: &CredentialsParserStub{},
	}
}

type CredentialsParserStub struct {
	errorStub error
}

func (p *CredentialsParserStub) ParseDatabaseCredentials() (database.DatabaseCredentials, error) {
	return database.DatabaseCredentials{}, p.errorStub
}

type ClientStub struct{}

func (c *ClientStub) CanRunRemoteCommand(command string) bool            { return true }
func (c *ClientStub) RunRemoteCommand(command string) (io.Reader, error) { return nil, nil }
func (c *ClientStub) Upload(r io.Reader, dst string) error               { return nil }
func (c *ClientStub) Delete(dst string) error                            { return nil }
func (c *ClientStub) Mkdir(dst string) error                             { return nil }
func (c *ClientStub) ReadDir(path string) ([]os.FileInfo, error)         { return nil, nil }
func (c *ClientStub) Open(path string) (*sftp.File, error)               { return nil, nil }
func (c *ClientStub) NewSession() (*ssh.Session, error)                  { return nil, nil }

type FileEmitterStub struct{}

func (e *FileEmitterStub) CalculateByteSize(src string) int                  { return 0 }
func (e *FileEmitterStub) EmitSingle(path string, fn emitter.EmitFunc) error { return nil }
func (e *FileEmitterStub) EmitAll(path string, fn emitter.EmitFunc) error    { return nil }

type HttpGetterStub struct{}

func (g *HttpGetterStub) Get(url string) (io.ReadCloser, error) { return nil, nil }
