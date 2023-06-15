package builder

import (
	"fmt"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"io"
	"os"
	"path/filepath"
)

func PackageWP(sshCredentials sftp.SSHCredentials, publicUrl, publicPath string) error {
	client, err := sftp.NewClient(sshCredentials)
	if err != nil {
		return fmt.Errorf("error creating new client: %w", err)
	}
	defer client.Close()

	operations, err := prepareOperations(client, publicUrl, PublicPath(publicPath))
	if err != nil {
		return fmt.Errorf("error preparing operations: %w", err)
	}

	builder := &Builder{PublicPath(publicPath), operations}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory: %w", err)
	}
	filename := filepath.Join(wd, "wp.zip")

	zipFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating zip file: %w", err)
	}
	defer zipFile.Close()

	err = builder.PackageWP(zipFile)
	if err != nil {
		return fmt.Errorf("error packaging WP: %w", err)
	}

	return nil
}

func prepareOperations(c *sftp.ClientWrapper, publicUrl string, pathToPublic PublicPath) ([]Operation, error) {
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

	generateJsonOperation := &GenerateJsonOperation{
		u:           c,
		g:           &BasicHttpGetter{},
		publicUrl:   publicUrl,
		publicPath:  pathToPublic,
		credentials: credentials,
	}

	operations := []Operation{
		downloadFilesOperation,
		exportDatabaseOperation,
		generateJsonOperation,
	}

	return operations, nil
}
