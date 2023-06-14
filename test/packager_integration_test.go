//go:build integration

package integration_test

import (
	"archive/zip"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jfortunato/wp-zip/internal/builder"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"log"
	"net"
	"os"
	"path/filepath"
	"testing"
)

// Docker will assign random ports to the container, and ory/dockertest allows us to retrieve the ports
// at runtime using something like resource.GetPort("internal_port/tcp")
var (
	SSH_PORT   string
	MYSQL_PORT string
)

// Assign some constants for the container
const (
	PATH_TO_DOCKERFILE  = "./docker/openssh-test-server/Dockerfile"
	CONTAINER_NAME      = "openssh-test-server"
	SSH_HOST            = "127.0.0.1"
	SSH_USER            = "test"
	SSH_PASS            = "test"
	MYSQL_ROOT_PASSWORD = "rootpass"
	MYSQL_DATABASE      = "some_db_name"
)

func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	// uses pool to try to connect to Docker
	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// we will use a dockerfile to build the image for testing
	resource, err := pool.BuildAndRunWithOptions(PATH_TO_DOCKERFILE, &dockertest.RunOptions{
		Name: CONTAINER_NAME,
		Env: []string{
			"MYSQL_ROOT_PASSWORD=" + MYSQL_ROOT_PASSWORD,
			"MYSQL_DATABASE=" + MYSQL_DATABASE,
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	resource.Expire(60) // Tell docker to hard kill the container in 60 seconds

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		SSH_PORT = resource.GetPort("22/tcp")
		MYSQL_PORT = resource.GetPort("3306/tcp")

		var err error

		// We want to ensure that both the ssh server and the database are up and running.
		// First, we check the ssh server.
		_, err = net.Dial("tcp", net.JoinHostPort("localhost", SSH_PORT))
		if err != nil {
			return err
		}

		// Then, we check the database.
		db, err := sql.Open("mysql", fmt.Sprintf("root:rootpass@(localhost:%s)/mysql", MYSQL_PORT))
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to either ssh or database: %s", err)
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestZipFileCreated(t *testing.T) {
	builder.PackageWP(sftp.SSHCredentials{User: SSH_USER, Pass: SSH_PASS, Host: SSH_HOST, Port: SSH_PORT}, "", "public")

	filename := outputFile()
	defer cleanup(t, filename)

	assertZipContainsFiles(t, filename, []string{"files/index.php", "files/wp-config.php", "database.sql"})
}

func outputFile() string {
	wd, _ := os.Getwd()

	return filepath.Join(wd, "wp.zip")
}

func cleanup(t *testing.T, zipFilename string) {
	t.Helper()

	// Delete the zip file
	err := os.Remove(zipFilename)
	if err != nil {
		t.Errorf("Error deleting zip file: %s", err)
	}
}

func assertZipContainsFiles(t *testing.T, filename string, files []string) {
	t.Helper()

	// extract the zip
	zipReader, err := zip.OpenReader(filename)
	if err != nil {
		t.Errorf("Error reading zip: %s", err)
	}

	// check that the zip contains the files we expect
	for _, file := range files {
		found := false
		for _, f := range zipReader.File {
			if f.Name == file {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected zip to contain file %s, but it did not", file)
		}
	}
}
