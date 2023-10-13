package database

import (
	"archive/zip"
	"embed"
	"errors"
	"fmt"
	"github.com/jfortunato/wp-zip/internal/emitter"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"github.com/jfortunato/wp-zip/internal/types"
	"io"
	"strings"
)

//go:embed assets
var assets embed.FS

const MYSQLDUMP_PHP_VERSION = "2.11"

// PHPDatabaseExporter is a DatabaseExporter that uses a custom PHP script to export the database. It should only be used as a fallback when the mysqldump command is not available.
// The script we utilize is bundled with this package to lock its version and ensure that it is always available. It's source can be found here:
// https://github.com/ifsnop/mysqldump-php
type PHPDatabaseExporter struct {
	u           sftp.FileUploadDeleter
	p           types.PublicPath
	siteUrl     types.SiteUrl
	g           HttpGetter
	e           emitter.FileEmitter
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
	resp, err := e.g.Get(string(e.siteUrl) + "/wp-zip-database-export/dump.php")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.New("invalid response from server"), err)
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
