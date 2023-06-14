package builder

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestExportDatabaseOperation_SendFiles(t *testing.T) {
	t.Run("it sends the remote database using the exporter", func(t *testing.T) {
		var tests = []struct {
			exporterResult   io.Reader
			expectedContents string
		}{
			{strings.NewReader("database.sql contents"), "database.sql contents"},
			{strings.NewReader("database.sql other contents"), "database.sql other contents"},
		}

		for _, test := range tests {
			operation := &ExportDatabaseOperation{&MockDatabaseExporter{contentsStub: test.exporterResult}}

			expectFilesSentFromOperation(t, operation, map[string]string{
				"database.sql": test.expectedContents,
			})
		}
	})

	t.Run("it returns an error if the exporter fails", func(t *testing.T) {
		operation := &ExportDatabaseOperation{&MockDatabaseExporter{errorStub: errors.New("exporter error")}}

		// Assert error returned
		err := operation.SendFiles(nil)
		if err == nil || err.Error() != "exporter error" {
			t.Errorf("operation.SendFiles() returned nil; want error")
		}

		// Assert no files sent
		expectFilesSentFromOperation(t, operation, map[string]string{})
	})
}

func TestMysqldumpDatabaseExporter_Export(t *testing.T) {
	t.Run("it returns an error if the remote server cannot run the mysqldump command", func(t *testing.T) {
		// The command runner will not be able to run the mysqldump command, so it will return an error
		commandRunner := &MockCommandRunner{}

		exporter := &MysqldumpDatabaseExporter{commandRunner, DatabaseCredentials{"User", "Pass", "Dbname"}}

		// Assert error returned
		_, err := exporter.Export()

		if err == nil || err.Error() != "mysqldump command not found" {
			t.Errorf("exporter.Export() returned nil; want error")
		}
	})

	t.Run("it returns an error if the credentials are incorrect", func(t *testing.T) {
		// The command runner will pass the version check, but fail the credential check
		commandRunner := &MockCommandRunner{map[string]string{"mysqldump --version": "mysqldump Ver 1.0"}}

		exporter := &MysqldumpDatabaseExporter{commandRunner, DatabaseCredentials{"User", "BadPass", "Dbname"}}

		// Assert error returned
		_, err := exporter.Export()

		if err == nil || err.Error() != "MySQL credentials are incorrect" {
			t.Errorf("exporter.Export() returned nil; want error")
		}
	})

	t.Run("it uses mysqldump to export to the reader", func(t *testing.T) {
		expectedOutput := "mysqldump Dbname output"

		commandRunner := &MockCommandRunner{commandsThatExist: map[string]string{
			"mysqldump --version":                             "mysqldump Ver 1.0",
			"mysql -uUser -pPass Dbname -e\"quit\"":           "",
			"mysqldump --no-tablespaces -uUser -pPass Dbname": expectedOutput,
		}}

		exporter := &MysqldumpDatabaseExporter{commandRunner, DatabaseCredentials{"User", "Pass", "Dbname"}}

		r, _ := exporter.Export()

		if readerToString(r) != expectedOutput {
			t.Errorf("exporter.Export() returned %s; want %s", r, expectedOutput)
		}
	})
}

type MockDatabaseExporter struct {
	contentsStub io.Reader
	errorStub    error
}

func (m *MockDatabaseExporter) Export() (io.Reader, error) {
	return m.contentsStub, m.errorStub
}

type MockCommandRunner struct {
	commandsThatExist map[string]string
}

func (m *MockCommandRunner) CanRunRemoteCommand(command string) bool {
	_, ok := m.commandsThatExist[command]
	return ok
}

func (m *MockCommandRunner) RunRemoteCommand(command string) (io.Reader, error) {
	return strings.NewReader(m.commandsThatExist[command]), nil
}