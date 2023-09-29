package database

import (
	"github.com/jfortunato/wp-zip/internal/emitter"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"github.com/jfortunato/wp-zip/internal/types"
	"io"
)

type DatabaseCredentials struct {
	User string
	Pass string
	Name string
}

type HttpGetter interface {
	Get(url string) (resp io.ReadCloser, err error)
}

type DatabaseExporter interface {
	Export() (io.Reader, error)
}

// NewDatabaseExporter is a factory function that returns a DatabaseExporter. It detects at runtime whether the remote server supports `mysqldump` or not, and returns the appropriate exporter.
func NewDatabaseExporter(c sftp.Client, p types.PublicPath, d types.Domain, g HttpGetter, e emitter.FileEmitter, creds DatabaseCredentials) DatabaseExporter {
	if c.CanRunRemoteCommand("mysqldump --version") {
		return &MysqldumpDatabaseExporter{c, creds}
	}

	return &PHPDatabaseExporter{c, p, d, g, e, creds}
}
