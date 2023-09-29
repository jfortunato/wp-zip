package packager

import (
	"errors"
	"github.com/jfortunato/wp-zip/internal/database"
	"github.com/jfortunato/wp-zip/internal/emitter"
	"github.com/jfortunato/wp-zip/internal/operations"
	"github.com/jfortunato/wp-zip/internal/sftp"
)

var ErrCannotParseCredentials = errors.New("cannot build - error parsing credentials")

// Builder is responsible for building the operations that will be run by the Runner.
type Builder struct {
	c sftp.Client
	e emitter.FileEmitter
	g operations.HttpGetter
	p CredentialsParser
}

type CredentialsParser interface {
	ParseDatabaseCredentials() (database.DatabaseCredentials, error)
}

func (b *Builder) Build(options Options) ([]operations.Operation, error) {
	credentials, err := b.p.ParseDatabaseCredentials()
	if err != nil {
		return nil, ErrCannotParseCredentials
	}

	return []operations.Operation{
		// The DownloadFilesOperation is responsible for downloading the entire site files from the server.
		operations.NewDownloadFilesOperation(b.e, options.PublicPath),
		// The ExportDatabaseOperation is responsible for exporting the database from the server.
		operations.NewExportDatabaseOperation(credentials, b.c, options.PublicPath, options.PublicUrl, b.g, b.e),
		// The GenerateJsonOperation is responsible for generating a JSON file containing metadata about the site (url, php version, etc).
		operations.NewGenerateJsonOperation(b.c, b.g, options.PublicUrl, options.PublicPath, credentials),
	}, nil
}
