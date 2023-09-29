package emitter

import (
	"github.com/jfortunato/wp-zip/internal/sftp"
	"io"
)

// A FileEmitter is basically a file downloader, but it doesn't actually download files to the filesystem. Instead, it just emits the file data (name, contents) and it's up to the caller to do something with it.
type FileEmitter interface {
	CalculateByteSize(src string) int
	EmitAll(src string, fn EmitFunc) error
	EmitSingle(src string, fn EmitFunc) error
}

type EmitFunc func(path string, contents io.Reader)

// NewFileEmitter is a factory function that returns a FileEmitter. It detects at runtime whether the remote server supports `tar` or not, and returns the appropriate downloader.
func NewFileEmitter(client sftp.Client) FileEmitter {
	if client.CanRunRemoteCommand("tar --version") {
		return &TarFileEmitter{client}
	}

	return &SftpFileEmitter{client}
}
