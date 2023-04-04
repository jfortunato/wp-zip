package packager

import (
	"fmt"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

// PackageWP is the main function that packages a WordPress site into a zip file
func PackageWP(credentials sftp.SSHCredentials, publicPath string) {
	// Assert that we can connect and the wp-config.php file exists under the publicPath
	// by reading it into a local string
	client, contents, err := setupClientAndReadWpConfig(credentials, filepath.Join(publicPath, "wp-config.php"))
	if err != nil {
		log.Fatalln(err)
	}
	defer client.Close()

	// Parse the wp-config.php file for the database credentials
	fields, err := parseWpConfig(contents)
	if err != nil {
		log.Fatalln(err)
	}

	// Create a temporary directory to store the files
	directory, err := os.MkdirTemp("", "wp-zip-")
	if err != nil {
		log.Fatalln(err)
	}
	filesDirectory := filepath.Join(directory, "files")
	os.Mkdir(filesDirectory, 0755)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Copying files to temporary directory: " + filesDirectory)
	err = client.DownloadDirectory(publicPath, filesDirectory)
	if err != nil {
		log.Fatalln(err)
	}

	databaseDirectory := filepath.Join(directory, "database")
	os.Mkdir(databaseDirectory, 0755)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Copying database to temporary directory: " + databaseDirectory)
	err = client.ExportDatabaseToFile(fields.dbUser, fields.dbPass, fields.dbName, filepath.Join(databaseDirectory, "database.sql"))
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Finished")
}

// This function isolates the sftp connection setup and ensures that a WordPress wp-config.php file
// exists at the specified path. It will return both the sftp client and the contents of the wp-config.php
// for convenience to use in the caller.
func setupClientAndReadWpConfig(credentials sftp.SSHCredentials, pathToWpConfig string) (*sftp.ClientWrapper, string, error) {
	client, err := sftp.NewClient(credentials)
	if err != nil {
		return nil, "", err
	}

	// Read the wp-config file
	contents, err := client.ReadFileToString(pathToWpConfig)
	if err != nil {
		client.Close()
		return nil, "", err
	}

	return client, contents, nil
}

type wpConfigFields struct {
	dbName string
	dbUser string
	dbPass string
}

func parseWpConfig(contents string) (wpConfigFields, error) {
	fields := map[string]string{
		"DB_NAME":     "",
		"DB_USER":     "",
		"DB_PASSWORD": "",
	}

	for field := range fields {
		value, err := parseWpConfigField(contents, field)
		if err != nil {
			return wpConfigFields{}, err
		}
		fields[field] = value
	}

	return wpConfigFields{fields["DB_NAME"], fields["DB_USER"], fields["DB_PASSWORD"]}, nil
}

func parseWpConfigField(contents, field string) (string, error) {
	re := regexp.MustCompile(`define\(['"]` + field + `['"], ['"](.*)['"]\);`)
	matches := re.FindStringSubmatch(contents)
	if len(matches) != 2 {
		return "", fmt.Errorf("could not parse %s from wp-config.php", field)
	}
	return matches[1], nil
}
