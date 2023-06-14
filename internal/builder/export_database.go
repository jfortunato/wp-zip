package builder

import (
	"errors"
	"io"
)

type ExportDatabaseOperation struct {
	exporter DatabaseExporter
}

type DatabaseExporter interface {
	Export() (io.Reader, error)
}

type DatabaseCredentials struct {
	User string
	Pass string
	Name string
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

type RemoteCommandRunner interface {
	CanRunRemoteCommand(command string) bool
	RunRemoteCommand(command string) (io.Reader, error)
}

type MysqldumpDatabaseExporter struct {
	commandRunner RemoteCommandRunner
	credentials   DatabaseCredentials
}

func (e *MysqldumpDatabaseExporter) Export() (io.Reader, error) {
	if !e.commandRunner.CanRunRemoteCommand("mysqldump --version") {
		return nil, errors.New("mysqldump command not found")
	}

	credentialsString := "-u" + e.credentials.User + " -p" + e.credentials.Pass + " " + e.credentials.Name

	if !e.commandRunner.CanRunRemoteCommand("mysql " + credentialsString + " -e\"quit\"") {
		return nil, errors.New("MySQL credentials are incorrect")
	}

	return e.commandRunner.RunRemoteCommand("mysqldump --no-tablespaces " + credentialsString)
}