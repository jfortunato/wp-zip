package operations

import (
	"github.com/jfortunato/wp-zip/internal/database"
	"github.com/jfortunato/wp-zip/internal/emitter"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"github.com/jfortunato/wp-zip/internal/types"
)

type ExportDatabaseOperation struct {
	exporter database.DatabaseExporter
}

func NewExportDatabaseOperation(credentials database.DatabaseCredentials, c sftp.Client, pathToPublic types.PublicPath, siteUrl types.SiteUrl, g HttpGetter, e emitter.FileEmitter) *ExportDatabaseOperation {
	exporter := database.NewDatabaseExporter(c, pathToPublic, siteUrl, g, e, credentials)

	return &ExportDatabaseOperation{exporter}
}

func (o *ExportDatabaseOperation) SendFiles(fn SendFilesFunc) error {
	r, err := o.exporter.Export()
	if err != nil {
		return err
	}

	return fn(File{
		Name: "database.sql",
		Body: r,
	})
}
