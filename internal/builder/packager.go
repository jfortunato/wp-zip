package builder

import (
	"fmt"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"io"
	"log"
	"os"
	"path/filepath"
)

func PackageWP(sshCredentials sftp.SSHCredentials, publicUrl, publicPath string) {
	client, err := sftp.NewClient(sshCredentials)
	if err != nil {
		log.Fatalln(err)
	}
	defer client.Close()

	operations, err := prepareOperations(client, PublicPath(publicPath))
	if err != nil {
		log.Fatalln(err)
	}

	builder := &Builder{PublicPath(publicPath), operations}
	if err != nil {
		log.Fatalln(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
	filename := filepath.Join(wd, "wp.zip")

	zipFile, err := os.Create(filename)
	if err != nil {
		log.Fatalln(err)
	}
	defer zipFile.Close()

	err = builder.PackageWP(zipFile)
	if err != nil {
		log.Fatalln(err)
	}
}

func prepareOperations(c *sftp.ClientWrapper, pathToPublic PublicPath) ([]Operation, error) {
	fileEmitter := initFileEmitter(c)

	downloadFilesOperation, err := NewDownloadFilesOperation(fileEmitter, pathToPublic)
	if err != nil {
		return nil, err
	}

	// Download/read the wp-config.php file
	var wpConfigFileContents string
	err = fileEmitter.EmitSingle(filepath.Join(string(pathToPublic), "wp-config.php"), func(path string, contents io.Reader) {
		wpConfigFileContents = readerToString(contents)
	})
	if err != nil {
		return nil, fmt.Errorf("error reading wp-config.php: %s", err)
	}
	if wpConfigFileContents == "" {
		return nil, fmt.Errorf("could not read wp-config.php")
	}
	credentials, err := parseDatabaseCredentials(wpConfigFileContents)
	if err != nil {
		return nil, fmt.Errorf("error parsing database credentials: %s", err)
	}

	exportDatabaseOperation := &ExportDatabaseOperation{&MysqldumpDatabaseExporter{
		c,
		credentials,
	}}

	operations := []Operation{
		downloadFilesOperation,
		exportDatabaseOperation,
	}

	return operations, nil
}
