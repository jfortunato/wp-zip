package builder

import (
	"github.com/pkg/sftp"
	"io"
	"os"
	"path/filepath"
)

type DownloadFilesOperation struct {
	emitter      FileEmitter
	pathToPublic PublicPath
}

type EmitFunc func(path string, contents io.Reader)

type FileEmitter interface {
	EmitAll(src string, fn EmitFunc) error
	EmitSingle(src string, fn EmitFunc) error
}

func NewDownloadFilesOperation(directoryEmitter FileEmitter, pathToPublic PublicPath) (*DownloadFilesOperation, error) {
	return &DownloadFilesOperation{directoryEmitter, pathToPublic}, nil
}

func (o *DownloadFilesOperation) SendFiles(fn SendFilesFunc) error {
	// Download the entire public directory and emit each file as they come in to the channel
	return o.emitter.EmitAll(string(o.pathToPublic), func(path string, contents io.Reader) {
		f := File{
			Name: path,
			Body: contents,
		}

		fn(f)
	})
}

// TarChecker is an interface that can be implemented by an sftp client to determine if the remote server supports `tar` or not.
type TarChecker interface {
	HasTar() bool
}

type Client interface {
	ReadDir(path string) ([]os.FileInfo, error)
	Open(path string) (*sftp.File, error)
}

// NewFileEmitter is a factory function that returns a FileEmitter. It detects at runtime whether the remote server supports `tar` or not, and returns the appropriate downloader.
func NewFileEmitter(checker TarChecker, client Client) FileEmitter {
	if checker.HasTar() {
		return &TarFileEmitter{}
	}

	return &SftpFileEmitter{client}
}

// TarFileEmitter runs `tar` on the remote server as an easy way to "stream" the entire directory at once, instead of opening and closing an SFTP connection for each file. This results in a much faster download, and is thr preferred method of downloading files.
type TarFileEmitter struct {
}

func (t *TarFileEmitter) EmitAll(src string, fn EmitFunc) error {
	return nil
}

func (t *TarFileEmitter) EmitSingle(src string, fn EmitFunc) error {
	return nil
}

// SftpFileEmitter downloads each file individually over SFTP. This is much slower than the TarFileEmitter, but is useful when the remote server doesn't have `tar` installed or otherwise doesn't support it.
type SftpFileEmitter struct {
	client Client
}

func (s *SftpFileEmitter) EmitAll(src string, fn EmitFunc) error {
	// Get the list of files in the remote directory
	remoteFiles, err := s.client.ReadDir(src)
	if err != nil {
		return err
	}
	for _, remoteFile := range remoteFiles {
		remoteFilepath := filepath.Join(src, remoteFile.Name())

		// If the file is a directory, recursively emit it
		if remoteFile.IsDir() {
			err := s.EmitAll(remoteFilepath, fn)
			if err != nil {
				return err
			}
		} else {
			// Otherwise, emit the file
			err := s.EmitSingle(remoteFilepath, fn)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SftpFileEmitter) EmitSingle(src string, fn EmitFunc) error {
	r, err := s.client.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	fn(src, r)

	return nil
}
