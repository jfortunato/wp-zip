package builder

import (
	"archive/zip"
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
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

// NewDatabaseExporter is a factory function that returns a DatabaseExporter. It detects at runtime whether the remote server supports `mysqldump` or not, and returns the appropriate exporter.
func NewDatabaseExporter(checker RemoteCommandRunner, u FileUploadDeleter, p PublicPath, d Domain, g HttpGetter, e FileEmitter, creds DatabaseCredentials) DatabaseExporter {
	if checker.CanRunRemoteCommand("mysqldump --version") {
		return &MysqldumpDatabaseExporter{checker, creds}
	}

	return &PHPDatabaseExporter{u, p, d, g, e, creds}
}

// MysqldumpDatabaseExporter is a DatabaseExporter that uses the mysqldump command to export the database. It should be the preferred exporter whenever possible.
type MysqldumpDatabaseExporter struct {
	commandRunner RemoteCommandRunner
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

// PHPDatabaseExporter is a DatabaseExporter that uses a custom PHP script to export the database. It should only be used as a fallback when the mysqldump command is not available.
// The script we utilize is bundled with this package to lock its version and ensure that it is always available. It's source can be found here:
// https://github.com/ifsnop/mysqldump-php
type PHPDatabaseExporter struct {
	u           FileUploadDeleter
	p           PublicPath
	publicUrl   Domain
	g           HttpGetter
	e           FileEmitter
	credentials DatabaseCredentials
}

// Export We need to upload the PHP script to the server, then run it to generate the database dump.
func (e *PHPDatabaseExporter) Export() (io.Reader, error) {
	// First, create a directory on the remote host to house our working directory
	dirName := e.p.String() + "wp-zip-database-export"
	err := e.u.Mkdir(dirName)
	if err != nil {
		return nil, err
	}
	defer e.deleteUploadedDir(dirName)

	// Next, upload the PHP script to the remote host
	dumper, err := extractPhpDumpScriptFromZip()
	if err != nil {
		return nil, err
	}
	defer dumper.Close()
	err = e.u.Upload(dumper, dirName+"/Mysqldump.php")
	if err != nil {
		return nil, err
	}
	// We also want to our short script that utilizes the dumper
	script := getPhpScriptContents(e.credentials)
	err = e.u.Upload(strings.NewReader(script), dirName+"/dump.php")
	if err != nil {
		return nil, err
	}

	// Finally, run the script on the remote host by making an HTTP request to it
	resp, err := e.g.Get(e.publicUrl.AsSecureUrl() + "/wp-zip-database-export/dump.php")
	if err != nil {
		// Try an insecure URL before returning an error
		resp, err = e.g.Get(e.publicUrl.AsInsecureUrl() + "/wp-zip-database-export/dump.php")
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidResponse, err)
		}
	}
	defer resp.Close()

	return resp, nil
}

func (e *PHPDatabaseExporter) deleteUploadedDir(dirName string) error {
	e.u.Delete(dirName + "/Mysqldump.php")
	e.u.Delete(dirName + "/dump.php")
	e.u.Delete(dirName + "/dump.sql")
	return e.u.Delete(dirName)
}

func ParseDatabaseCredentials(contents string) (DatabaseCredentials, error) {
	var fields = map[string]string{"DB_USER": "", "DB_PASSWORD": "", "DB_NAME": ""}

	for field, _ := range fields {
		value, err := parseWpConfigField(contents, field)
		if err != nil {
			return DatabaseCredentials{}, err
		}
		fields[field] = value
	}

	return DatabaseCredentials{User: fields["DB_USER"], Pass: fields["DB_PASSWORD"], Name: fields["DB_NAME"]}, nil
}

func parseWpConfigField(contents, field string) (string, error) {
	re := regexp.MustCompile(`define\( *['"]` + field + `['"], *['"](.*)['"] *\);`)
	matches := re.FindStringSubmatch(contents)
	if len(matches) != 2 {
		return "", fmt.Errorf("could not parse %s from wp-config.php", field)
	}
	return matches[1], nil
}

//go:embed assets
var assets embed.FS

const MYSQLDUMP_PHP_VERSION = "2.11"

func extractPhpDumpScriptFromZip() (io.ReadCloser, error) {
	zf, err := assets.ReadFile("assets/mysqldump-php-" + MYSQLDUMP_PHP_VERSION + ".zip")
	if err != nil {
		return nil, err
	}

	strZip := string(zf)
	r, err := zip.NewReader(strings.NewReader(strZip), int64(len(strZip)))
	if err != nil {
		return nil, err
	}

	f, err := r.Open("mysqldump-php-" + MYSQLDUMP_PHP_VERSION + "/src/Ifsnop/Mysqldump/Mysqldump.php")
	if err != nil {
		return nil, err
	}

	return f, nil
}

func getPhpScriptContents(creds DatabaseCredentials) string {
	return fmt.Sprintf(`<?php

include_once(dirname(__FILE__) . '/Mysqldump.php');
$dump = new Ifsnop\Mysqldump\Mysqldump('mysql:host=localhost;dbname=%s', '%s', '%s');
$dump->start(dirname(__FILE__) . '/dump.sql');
$contents = file_get_contents(dirname(__FILE__) . '/dump.sql');
echo $contents;
`, creds.Name, creds.User, creds.Pass)
}
