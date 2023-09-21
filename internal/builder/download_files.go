package builder

import (
	"archive/tar"
	"github.com/pkg/sftp"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type DownloadFilesOperation struct {
	emitter      FileEmitter
	pathToPublic PublicPath
}

type EmitFunc func(path string, contents io.Reader)

// A FileEmitter is basically a file downloader, but it doesn't actually download files to the filesystem. Instead, it just emits the file data (name, contents) and it's up to the caller to do something with it.
type FileEmitter interface {
	CalculateByteSize(src string) int
	EmitAll(src string, fn EmitFunc) error
	EmitSingle(src string, fn EmitFunc) error
}

func NewDownloadFilesOperation(directoryEmitter FileEmitter, pathToPublic PublicPath) (*DownloadFilesOperation, error) {
	return &DownloadFilesOperation{directoryEmitter, pathToPublic}, nil
}

func (o *DownloadFilesOperation) SendFiles(fn SendFilesFunc) error {
	bar := progressbar.DefaultBytes(int64(o.emitter.CalculateByteSize(string(o.pathToPublic))), "Downloading files")
	defer bar.Clear()

	// Download the entire public directory and emit each file as they come in to the channel
	return o.emitter.EmitAll(string(o.pathToPublic), func(path string, contents io.Reader) {
		f := File{
			Name: filepath.Join("files", path), // We want to store the files in the "files" directory
			// The progress bar is a writer, so we can write to it to update the progress
			Body: io.TeeReader(contents, bar),
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

func (t *TarFileEmitter) CalculateByteSize(src string) int {
	sess, err := t.client.NewSession()
	if err != nil {
		log.Fatalln("failed to create session: %w", err)
	}
	defer sess.Close()

	// Determine the total size of the directory in bytes
	res, err := sess.Output("du -sb " + filepath.Dir(src) + " | awk '{print $1}'")
	if err != nil {
		log.Fatalln("failed to run find: %w", err)
	}

	// Convert the byte slice to a string, then convert the string to an int
	numFiles, err := strconv.Atoi(strings.TrimSpace(string(res)))
	if err != nil {
		log.Fatalln("failed to convert string to int: %w", err)
	}

	return numFiles
}

func (t *TarFileEmitter) EmitAll(src string, fn EmitFunc) error {
	return t.emit(src, ".", fn)
}

func (t *TarFileEmitter) EmitSingle(src string, fn EmitFunc) error {
	return t.emit(filepath.Dir(src), filepath.Base(src), fn)
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

func (s *SftpFileEmitter) CalculateByteSize(src string) int {
	// Takes too long to calculate the size of the directory, so just return -1 which indicates
	// to the progress bar to use an indeterminate spinner
	return -1
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
