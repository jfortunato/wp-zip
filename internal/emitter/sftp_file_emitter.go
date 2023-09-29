package emitter

import "github.com/jfortunato/wp-zip/internal/sftp"

// SftpFileEmitter downloads each file individually over SFTP. This is much slower than the TarFileEmitter, but is useful when the remote server doesn't have `tar` installed or otherwise doesn't support it.
type SftpFileEmitter struct {
	r sftp.RemoteFileReader
}

func (s *SftpFileEmitter) CalculateByteSize(src string) int {
	// Takes too long to calculate the size of the directory, so just return -1 which indicates
	// to the progress bar to use an indeterminate spinner
	return -1
}

func (s *SftpFileEmitter) EmitAll(src string, fn EmitFunc) error {
	// Get the list of files in the remote directory
	remoteFiles, err := s.r.ReadDir(src)
	if err != nil {
		return err
	}
	for _, remoteFile := range remoteFiles {
		remoteFilepath := src + "/" + remoteFile.Name()

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
	r, err := s.r.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	fn(src, r)

	return nil
}
