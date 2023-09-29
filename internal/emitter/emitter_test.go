package emitter

import (
	_sftp "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"reflect"
	"testing"
)

func TestNewFileEmitter(t *testing.T) {
	t.Run("it should return the proper emitter depending on the client's tar support", func(t *testing.T) {
		var tests = []struct {
			isTarSupported bool
			expectedType   string
		}{
			{isTarSupported: true, expectedType: "*emitter.TarFileEmitter"},
			{isTarSupported: false, expectedType: "*emitter.SftpFileEmitter"},
		}

		for _, test := range tests {
			supportedCommands := map[string]string{}
			if test.isTarSupported {
				supportedCommands["tar --version"] = "tar version 1.0.0"
			}

			emitter := NewFileEmitter(&ClientStub{supportedCommands: supportedCommands})

			if reflect.TypeOf(emitter).String() != test.expectedType {
				t.Errorf("got type %s; want %s", reflect.TypeOf(emitter).String(), test.expectedType)
			}
		}
	})
}

type ClientStub struct {
	supportedCommands map[string]string
}

func (c *ClientStub) CanRunRemoteCommand(command string) bool {
	_, ok := c.supportedCommands[command]
	return ok
}
func (c *ClientStub) RunRemoteCommand(command string) (io.Reader, error) { return nil, nil }
func (c *ClientStub) Upload(r io.Reader, dst string) error               { return nil }
func (c *ClientStub) Delete(dst string) error                            { return nil }
func (c *ClientStub) Mkdir(dst string) error                             { return nil }
func (c *ClientStub) ReadDir(path string) ([]os.FileInfo, error)         { return nil, nil }
func (c *ClientStub) Open(path string) (*_sftp.File, error)              { return nil, nil }
func (c *ClientStub) NewSession() (*ssh.Session, error)                  { return nil, nil }
