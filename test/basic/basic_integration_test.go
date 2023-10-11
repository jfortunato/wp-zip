//go:build integration

package basic_test

import (
	"github.com/jfortunato/wp-zip/internal/packager"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"github.com/jfortunato/wp-zip/internal/types"
	"github.com/jfortunato/wp-zip/test"
	"os"
	"path/filepath"
	"testing"
)

// Assign some constants for the container
const (
	PATH_TO_DOCKERFILE  = "./docker/openssh-test-server/Dockerfile"
	SSH_HOST            = "127.0.0.1"
	SSH_USER            = "test"
	SSH_PASS            = "test"
	MYSQL_ROOT_PASSWORD = "rootpass"
	MYSQL_DATABASE      = "some_db_name"
	DOCUMENT_ROOT       = "/var/www/html"
)

func TestZipFileCreated(t *testing.T) {
	container := test.StartContainer(t, test.DefaultContainerRequest(filepath.Dir(PATH_TO_DOCKERFILE), MYSQL_ROOT_PASSWORD, MYSQL_DATABASE))

	domain := types.Domain("localhost:" + container.MappedPort("80/tcp"))

	filename := filepath.Join(os.TempDir(), "wp-zip-basic.zip")
	defer cleanup(t, filename)

	p, _ := packager.NewPackager(sftp.SSHCredentials{User: SSH_USER, Pass: SSH_PASS, Host: SSH_HOST, Port: container.MappedPort("22/tcp")}, domain, DOCUMENT_ROOT)
	_ = p.PackageWP(filename)

	test.AssertZipContainsFiles(t, filename, []string{"files/index.php", "files/wp-config.php", "database.sql", "wpmigrate-export.json"})
}

func TestUploadedFileIsAlwaysDeleted(t *testing.T) {
	container := test.StartContainer(t, test.DefaultContainerRequest(filepath.Dir(PATH_TO_DOCKERFILE), MYSQL_ROOT_PASSWORD, MYSQL_DATABASE))

	// When an invalid url is passed to the builder, it runs successfully up until it needs to generate the json file
	// and send an http request.
	invalidDomain := types.Domain("localhost:8888")

	credentials := sftp.SSHCredentials{User: SSH_USER, Pass: SSH_PASS, Host: SSH_HOST, Port: container.MappedPort("22/tcp")}

	filename := filepath.Join(os.TempDir(), "wp-zip-basic-deleted.zip")
	defer cleanup(t, filename)

	// We expect an error here because the url is invalid
	p, _ := packager.NewPackager(credentials, invalidDomain, DOCUMENT_ROOT)
	err := p.PackageWP(filename)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	test.AssertRemoteFileDoesNotExist(t, credentials, DOCUMENT_ROOT, `wp-zip-[^.]+\.php`)
}

func cleanup(t *testing.T, zipFilename string) {
	t.Helper()

	// Delete the zip file
	err := os.Remove(zipFilename)
	if err != nil {
		t.Errorf("Error deleting zip file: %s", err)
	}
}
