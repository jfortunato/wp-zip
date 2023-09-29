package database

import (
	"errors"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"io"
	"strings"
)

// MysqldumpDatabaseExporter is a DatabaseExporter that uses the mysqldump command to export the database. It should be the preferred exporter whenever possible.
type MysqldumpDatabaseExporter struct {
	commandRunner sftp.RemoteCommandRunner
	credentials   DatabaseCredentials
}

func (e *MysqldumpDatabaseExporter) Export() (io.Reader, error) {
	if !e.commandRunner.CanRunRemoteCommand("mysqldump --version") {
		return nil, errors.New("mysqldump command not found")
	}

	// Since the credentials are wrapped in single quotes, we need to replace any single quotes in the credentials.
	// For example, if the password is "password'123", we need to replace the single quote with '\'' so that the command
	// looks like this: -p'password'\''123'
	pass := strings.ReplaceAll(e.credentials.Pass, "'", `'\''`)

	credentialsString := "-u'" + e.credentials.User + "' -p'" + pass + "' " + e.credentials.Name

	if !e.commandRunner.CanRunRemoteCommand("mysql " + credentialsString + ` -e"quit"`) {
		return nil, errors.New("MySQL credentials are incorrect")
	}

	return e.commandRunner.RunRemoteCommand("mysqldump --no-tablespaces " + credentialsString)
}
