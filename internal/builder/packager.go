package builder

import (
	"fmt"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"io"
	"os"
)

func PackageWP(sshCredentials sftp.SSHCredentials, publicUrl Domain, publicPath PublicPath, outputFilename string) error {
	client, err := sftp.NewClient(sshCredentials)
	if err != nil {
		return fmt.Errorf("error creating new client: %w", err)
	}
	defer client.Close()

	operations, err := prepareOperations(client, publicUrl, publicPath)
	if err != nil {
		return fmt.Errorf("error preparing operations: %w", err)
	}

	builder := &Builder{publicPath, operations}

	zipFile, err := os.Create(outputFilename)
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

func prepareOperations(c *sftp.ClientWrapper, publicUrl Domain, pathToPublic PublicPath) ([]Operation, error) {
	fileEmitter := initFileEmitter(c)

	downloadFilesOperation, err := NewDownloadFilesOperation(fileEmitter, pathToPublic)
	if err != nil {
		return nil, err
	}

	// Download/read the wp-config.php file
	var wpConfigFileContents string
	err = fileEmitter.EmitSingle(string(pathToPublic)+"/wp-config.php", func(path string, contents io.Reader) {
		wpConfigFileContents = readerToString(contents)
	})
	if err != nil {
		return nil, fmt.Errorf("error reading wp-config.php: %s", err)
	}
	if wpConfigFileContents == "" {
		return nil, fmt.Errorf("could not read wp-config.php")
	}

	credentials, err := ParseDatabaseCredentials(wpConfigFileContents)
	if err != nil {
		return nil, fmt.Errorf("error parsing database credentials: %s", err)
	}

	g := &BasicHttpGetter{}

	exportDatabaseOperation := &ExportDatabaseOperation{NewDatabaseExporter(c, c, pathToPublic, publicUrl, g, fileEmitter, credentials)}

	generateJsonOperation := &GenerateJsonOperation{
		u:           c,
		g:           g,
		publicUrl:   publicUrl,
		publicPath:  pathToPublic,
		credentials: credentials,
		randomFileNamer: func() string {
			return "wp-zip-" + randSeq(10) + ".php"
		},
	}

	operations := []Operation{
		downloadFilesOperation,
		exportDatabaseOperation,
		generateJsonOperation,
	}

	return operations, nil
}
