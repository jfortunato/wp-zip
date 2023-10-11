package test

import (
	"archive/zip"
	"fmt"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"io"
	"regexp"
	"strings"
	"testing"
)

func AssertZipContainsFiles(t *testing.T, filename string, files []string) {
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

func AssertRemoteFileDoesNotExist(t *testing.T, credentials sftp.SSHCredentials, directory, regex string) {
	t.Helper()

	// connect to the server
	client, err := sftp.NewClient(credentials)
	if err != nil {
		t.Errorf("Error connecting to server: %s", err)
	}
	defer client.Close()

	// check that the file does not exist
	ok := client.CanRunRemoteCommand(fmt.Sprintf("find %s | grep -P '%s'", directory, regex))
	if ok {
		t.Errorf("Expected file to not exist, but it did")
	}
}

func AssertFileContainsMatch(t *testing.T, zipFilename, fileInZip, regex string) {
	t.Helper()

	// extract the zip
	zipReader, err := zip.OpenReader(zipFilename)
	if err != nil {
		t.Errorf("Error reading zip: %s", err)
	}

	// check that the zip contains the files we expect
	for _, f := range zipReader.File {
		if f.Name == fileInZip {
			// open the file
			rc, err := f.Open()
			if err != nil {
				t.Errorf("Error opening file: %s", err)
			}
			defer rc.Close()

			// read the file
			buf := new(strings.Builder)
			_, err = io.Copy(buf, rc)
			if err != nil {
				t.Errorf("Error reading file: %s", err)
			}
			contents := buf.String()

			// check that the file contains the regex
			ok, err := regexp.MatchString(regex, contents)
			if err != nil {
				t.Errorf("Error matching regex: %s", err)
			}
			if !ok {
				t.Errorf("Expected file to match regex, but it did not")
			}
		}
	}
}
