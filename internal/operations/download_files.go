package operations

import (
	"github.com/jfortunato/wp-zip/internal/emitter"
	"github.com/jfortunato/wp-zip/internal/types"
	"github.com/schollz/progressbar/v3"
	"io"
	"strings"
)

type DownloadFilesOperation struct {
	emitter      emitter.FileEmitter
	pathToPublic types.PublicPath
}

func NewDownloadFilesOperation(directoryEmitter emitter.FileEmitter, pathToPublic types.PublicPath) *DownloadFilesOperation {
	return &DownloadFilesOperation{directoryEmitter, pathToPublic}
}

func (o *DownloadFilesOperation) SendFiles(fn SendFilesFunc) error {
	bar := progressbar.DefaultBytes(int64(o.emitter.CalculateByteSize(string(o.pathToPublic))), "Downloading files")
	defer bar.Clear()

	// Download the entire public directory and emit each file as they come in to the channel
	return o.emitter.EmitAll(string(o.pathToPublic), func(path string, contents io.Reader) {
		// Remove the leading pathToPublic from the path
		path = strings.TrimPrefix(path, o.pathToPublic.String())

		f := File{
			Name: "files/" + path, // We want to store the files in the "files" directory
			// The progress bar is a writer, so we can write to it to update the progress
			Body: io.TeeReader(contents, bar),
		}

		fn(f)
	})
}
