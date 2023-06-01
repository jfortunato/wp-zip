package builder

import "io"

type DownloadFilesOperation struct {
	downloader   DirectoryDownloader
	pathToPublic PublicPath
}

type DownloadFunc func(path string, contents io.Reader)

type DirectoryDownloader interface {
	Download(src string, fn DownloadFunc) error
}

func NewDownloadFilesOperation(directoryDownloader DirectoryDownloader, pathToPublic PublicPath) (*DownloadFilesOperation, error) {
	return &DownloadFilesOperation{directoryDownloader, pathToPublic}, nil
}

func (o *DownloadFilesOperation) SendFiles(fn SendFilesFunc) error {
	// Download the entire public directory and emit each file as they come in to the channel
	return o.downloader.Download(string(o.pathToPublic), func(path string, contents io.Reader) {
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

// NewDirectoryDownloader is a factory function that returns a DirectoryDownloader. It detects at runtime whether the remote server supports `tar` or not, and returns the appropriate downloader.
func NewDirectoryDownloader(checker TarChecker) DirectoryDownloader {
	if checker.HasTar() {
		return &TarDirectoryDownloader{}
	}

	return &SftpDirectoryDownloader{}
}

// TarDirectoryDownloader runs `tar` on the remote server as an easy way to "stream" the entire directory at once, instead of opening and closing an SFTP connection for each file. This results in a much faster download, and is thr preferred method of downloading files.
type TarDirectoryDownloader struct {
}

func (t *TarDirectoryDownloader) Download(src string, fn DownloadFunc) error {
	return nil
}

// SftpDirectoryDownloader downloads each file individually over SFTP. This is much slower than the TarDirectoryDownloader, but is useful when the remote server doesn't have `tar` installed or otherwise doesn't support it.
type SftpDirectoryDownloader struct {
}

func (s *SftpDirectoryDownloader) Download(src string, fn DownloadFunc) error {
	return nil
}
