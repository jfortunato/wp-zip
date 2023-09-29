package database

import (
	"io"
	"strings"
	"testing"
)

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
		var tests = []struct {
			name     string
			creds    DatabaseCredentials
			checkCmd string
			dumpCmd  string
		}{
			{
				"no special chars",
				DatabaseCredentials{"User", "Pass", "Dbname"},
				`mysql -u'User' -p'Pass' Dbname -e"quit"`,
				"mysqldump --no-tablespaces -u'User' -p'Pass' Dbname",
			},
			{
				"single quote in middle of password",
				DatabaseCredentials{"User", "Pa'ss", "Dbname"},
				`mysql -u'User' -p'Pa'\''ss' Dbname -e"quit"`,
				`mysqldump --no-tablespaces -u'User' -p'Pa'\''ss' Dbname`,
			},
			{
				"single quote at beginning of password",
				DatabaseCredentials{"User", "'Pass", "Dbname"},
				`mysql -u'User' -p''\''Pass' Dbname -e"quit"`,
				`mysqldump --no-tablespaces -u'User' -p''\''Pass' Dbname`,
			},
			{
				"single quote at end of password",
				DatabaseCredentials{"User", "Pass'", "Dbname"},
				`mysql -u'User' -p'Pass'\''' Dbname -e"quit"`,
				`mysqldump --no-tablespaces -u'User' -p'Pass'\''' Dbname`,
			},
			{
				"double quote in middle of password",
				DatabaseCredentials{"User", `Pa"ss`, "Dbname"},
				`mysql -u'User' -p'Pa"ss' Dbname -e"quit"`,
				`mysqldump --no-tablespaces -u'User' -p'Pa"ss' Dbname`,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				expectedOutput := "mysqldump Dbname output"

				commandRunner := &MockCommandRunner{commandsThatExist: map[string]string{
					"mysqldump --version": "mysqldump Ver 1.0",
					test.checkCmd:         "",
					test.dumpCmd:          expectedOutput,
				}}

				exporter := &MysqldumpDatabaseExporter{commandRunner, test.creds}

				r, _ := exporter.Export()

				if readerToString(r) != expectedOutput {
					t.Errorf("exporter.Export() returned %s; want %s", r, expectedOutput)
				}
			})
		}
	})
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
