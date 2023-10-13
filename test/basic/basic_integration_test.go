//go:build acceptance

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
	PATH_TO_COMPOSE_FILE = "./docker/openssh-test-server/docker-compose.yml"
	SSH_HOST             = "127.0.0.1"
	SSH_USER             = "test"
	SSH_PASS             = "test"
	DOCUMENT_ROOT        = "/var/www/html"
)

func TestZipFileCreated(t *testing.T) {
	containers := test.StartComposeContainers(t, test.DefaultComposeRequest(PATH_TO_COMPOSE_FILE))

	url, _ := types.NewSiteUrl("http://localhost:" + containers["wordpress"].MappedPort("80/tcp"))

	test.InstallWP(t, containers["wordpress"], url)

	filename := filepath.Join(os.TempDir(), "wp-zip-basic.zip")
	defer cleanup(t, filename)

	p, _ := packager.NewPackager(sftp.SSHCredentials{User: SSH_USER, Pass: SSH_PASS, Host: SSH_HOST, Port: containers["wordpress"].MappedPort("22/tcp")}, url, DOCUMENT_ROOT)
	_ = p.PackageWP(filename)

	test.AssertZipContainsFiles(t, filename, []string{"files/index.php", "files/wp-config.php", "database.sql", "wpmigrate-export.json"})
	test.AssertFileContainsMatch(t, filename, "wpmigrate-export.json", `"name":"`+url.Domain()+`"`)
}

func TestUploadedFileIsAlwaysDeleted(t *testing.T) {
	containers := test.StartComposeContainers(t, test.DefaultComposeRequest(PATH_TO_COMPOSE_FILE))

	// When an invalid url is passed to the builder, it runs successfully up until it needs to generate the json file
	// and send an http request.
	invalidDomain := types.SiteUrl("http://localhost:8888")

	test.InstallWP(t, containers["wordpress"], invalidDomain)

	credentials := sftp.SSHCredentials{User: SSH_USER, Pass: SSH_PASS, Host: SSH_HOST, Port: containers["wordpress"].MappedPort("22/tcp")}

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

func TestDomainCanBeInferredFromDatabase(t *testing.T) {
	containers := test.StartComposeContainers(t, test.DefaultComposeRequest(PATH_TO_COMPOSE_FILE))

	expectedUrl, _ := types.NewSiteUrl("http://localhost:" + containers["wordpress"].MappedPort("80/tcp"))

	test.InstallWP(t, containers["wordpress"], expectedUrl)

	filename := filepath.Join(os.TempDir(), "wp-zip-basic-detect-domain.zip")
	defer cleanup(t, filename)

	p, _ := packager.NewPackager(sftp.SSHCredentials{User: SSH_USER, Pass: SSH_PASS, Host: SSH_HOST, Port: containers["wordpress"].MappedPort("22/tcp")}, "", DOCUMENT_ROOT)
	_ = p.PackageWP(filename)

	test.AssertZipContainsFiles(t, filename, []string{"files/index.php", "files/wp-config.php", "database.sql", "wpmigrate-export.json"})
	test.AssertFileContainsMatch(t, filename, "wpmigrate-export.json", `"domain":"`+expectedUrl.Domain()+`"`)
}

func cleanup(t *testing.T, zipFilename string) {
	t.Helper()

	// Delete the zip file
	err := os.Remove(zipFilename)
	if err != nil {
		t.Errorf("Error deleting zip file: %s", err)
	}
}
