package emitter

import (
	"archive/tar"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"io"
	"log"
	"strconv"
	"strings"
)

// TarFileEmitter runs `tar` on the remote server as an easy way to "stream" the entire directory at once, instead of opening and closing an SFTP connection for each file. This results in a much faster download, and is the preferred method of downloading files.
type TarFileEmitter struct {
	r sftp.RemoteFileReader
}

func (t *TarFileEmitter) CalculateByteSize(src string) int {
	sess, err := t.r.NewSession()
	if err != nil {
		log.Fatalln("failed to create session: %w", err)
	}
	defer sess.Close()

	// Determine the total size of the directory in bytes
	res, err := sess.Output("du -sb " + src + " | awk '{print $1}'")
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
	parentDirectory, filepathRelativeToParent := separateParentFromFilename(src)

	return t.emit(parentDirectory, filepathRelativeToParent, fn)
}

func (t *TarFileEmitter) emit(parentDirectory, filepathRelativeToParent string, fn EmitFunc) error {
	// We'll pipe the remote tar output directly into the tar reader
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		sess, err := t.r.NewSession()
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

		targetPath := strings.Join(paths, "/")

		// If the file is a directory, we don't need to do anything
		if header.FileInfo().IsDir() {
			continue
		}

		// Emit the file
		fn(targetPath, tr)
	}

	return nil
}

// Helper function that returns the parent directory and the basename of the file similar
// to filepath.Dir() and filepath.Base(), but always assumes unix-style paths.
func separateParentFromFilename(src string) (string, string) {
	paths := strings.Split(src, "/")
	parentDirectory := strings.Join(paths[:len(paths)-1], "/")
	filepathRelativeToParent := paths[len(paths)-1]

	return parentDirectory, filepathRelativeToParent
}
