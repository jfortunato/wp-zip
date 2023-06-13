package builder

import (
	"archive/tar"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type DownloadFilesOperation struct {
	emitter      FileEmitter
	pathToPublic PublicPath
}

type EmitFunc func(path string, contents io.Reader)

// A FileEmitter is basically a file downloader, but it doesn't actually download files to the filesystem. Instead, it just emits the file data (name, contents) and it's up to the caller to do something with it.
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
			Name: filepath.Join("files", path), // We want to store the files in the "files" directory
			Body: contents,
		}

		fn(f)
	})
}

type Client interface {
	ReadDir(path string) ([]os.FileInfo, error)
	Open(path string) (*sftp.File, error)
	NewSession() (*ssh.Session, error)
}

// NewFileEmitter is a factory function that returns a FileEmitter. It detects at runtime whether the remote server supports `tar` or not, and returns the appropriate downloader.
func NewFileEmitter(checker RemoteCommandRunner, client Client) FileEmitter {
	if checker.CanRunRemoteCommand("tar --version") {
		return &TarFileEmitter{client}
	}

	return &SftpFileEmitter{client}
}

// TarFileEmitter runs `tar` on the remote server as an easy way to "stream" the entire directory at once, instead of opening and closing an SFTP connection for each file. This results in a much faster download, and is the preferred method of downloading files.
type TarFileEmitter struct {
	client Client
}

func (t *TarFileEmitter) EmitAll(src string, fn EmitFunc) error {
	return t.emit(src, ".", fn)
}

func (t *TarFileEmitter) EmitSingle(src string, fn EmitFunc) error {
	paths := strings.Split(src, "/")

	parentDirectory := strings.Join(paths[:len(paths)-1], "/")
	filepathRelativeToParent := paths[len(paths)-1]

	return t.emit(parentDirectory, filepathRelativeToParent, fn)
}

func (t *TarFileEmitter) emit(parentDirectory, filepathRelativeToParent string, fn EmitFunc) error {
	// We'll pipe the remote tar output directly into the tar reader
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		sess, err := t.client.NewSession()
		if err != nil {
			log.Fatalln("failed to create session: %w", err)
		}
		sess.Stdout = writer
		defer sess.Close()

		if err := sess.Run("tar -C " + parentDirectory + " -cf - " + filepathRelativeToParent); err != nil {
			log.Fatalln("failed to run tar: %w", err)
		}
	}()

	tr := tar.NewReader(reader)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Remove any trailing slashes
		header.Name = strings.TrimSuffix(header.Name, "/")
		// Split the path into its components
		paths := strings.Split(header.Name, "/")

		if filepathRelativeToParent == "." {
			// Ignore the first path (name should be something like "./foo, so disregard the ".)
			paths = paths[1:]
			// Don't do anything for the top level directory
			if len(paths) == 0 {
				continue
			}
		}

		targetPath := filepath.Join(paths...)

		// If the file is a directory, we don't need to do anything
		if header.FileInfo().IsDir() {
			continue
		}

		// Emit the file
		fn(targetPath, tr)
	}

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
