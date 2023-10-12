package packager

import (
	"github.com/jfortunato/wp-zip/internal/emitter"
	"github.com/jfortunato/wp-zip/internal/operations"
	"github.com/jfortunato/wp-zip/internal/sftp"
)

// Builder is responsible for building the operations that will be run by the Runner.
type Builder struct {
	c sftp.Client
	e emitter.FileEmitter
	g operations.HttpGetter
}

func (b *Builder) Build(info SiteInfo) ([]operations.Operation, error) {
	return []operations.Operation{
		// The DownloadFilesOperation is responsible for downloading the entire site files from the server.
		operations.NewDownloadFilesOperation(b.e, info.publicPath),
		// The ExportDatabaseOperation is responsible for exporting the database from the server.
		operations.NewExportDatabaseOperation(info.dbCredentials, b.c, info.publicPath, info.siteUrl, b.g, b.e),
		// The GenerateJsonOperation is responsible for generating a JSON file containing metadata about the site (url, php version, etc).
		operations.NewGenerateJsonOperation(b.c, b.g, info.siteUrl, info.publicPath, info.dbCredentials),
	}, nil
}
