package packager

import (
	"fmt"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

// PackageWP is the main function that packages a WordPress site into a zip file
func PackageWP(credentials sftp.SSHCredentials, publicUrl, publicPath string) {
	// Assert that we can connect and the wp-config.php file exists under the publicPath
	// by reading it into a local string
	client, contents, err := setupClientAndReadWpConfig(credentials, publicPath+"/wp-config.php") // Assume a unix-like path on the server
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
	// Delete the temporary directory when we're done
	defer os.RemoveAll(directory)

	// Run downloadFiles and downloadDatabase in go routines, and wait for them to finish
	// before moving on to the next step
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := downloadFiles(client, directory, publicPath); err != nil {
			log.Fatalln(err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := downloadDatabase(client, directory, fields); err != nil {
			log.Fatalln(err)
		}
	}()
	wg.Wait()

	if err := generateJsonFile(client, directory, fields, publicUrl, publicPath); err != nil {
		log.Fatalln(err)
	}
	if err := zipFiles(client, directory); err != nil {
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

func downloadFiles(client *sftp.ClientWrapper, directory, publicPath string) error {
	filesDirectory := filepath.Join(directory, "files")

	if err := os.Mkdir(filesDirectory, 0755); err != nil {
		return fmt.Errorf("could not create files directory: %w", err)
	}

	log.Println("Copying files to temporary directory: " + filesDirectory)

	if err := client.DownloadDirectory(publicPath, filesDirectory); err != nil {
		return fmt.Errorf("could not download files: %w", err)
	}

	log.Println("Finished copying files")

	return nil
}

func downloadDatabase(client *sftp.ClientWrapper, directory string, fields wpConfigFields) error {
	log.Println("Copying database to temporary directory: " + directory)

	if err := client.ExportDatabaseToFile(fields.dbUser, fields.dbPass, fields.dbName, filepath.Join(directory, "database.sql")); err != nil {
		return fmt.Errorf("could not download database: %w", err)
	}

	log.Println("Finished copying database")

	return nil
}

func generateJsonFile(client *sftp.ClientWrapper, directory string, fields wpConfigFields, publicUrl, publicPath string) error {
	log.Println("Generating JSON file")

	publicUrl = fmt.Sprintf("https://%s", publicUrl)

	if err := client.GenerateJsonFile(fields.dbUser, fields.dbPass, fields.dbName, publicUrl, publicPath, filepath.Join(directory, "wpmigrate-export.json")); err != nil {
		return fmt.Errorf("could not generate JSON file: %w", err)
	}

	log.Println("Finished generating JSON file")

	return nil
}

func zipFiles(client *sftp.ClientWrapper, directory string) error {
	log.Println("Zipping files")

	wd, err := os.Getwd()

	if err != nil {
		return fmt.Errorf("could not get working directory: %w", err)
	}

	if err = client.WriteZip(directory, filepath.Join(wd, "wp.zip")); err != nil {
		return fmt.Errorf("could not zip files: %w", err)
	}

	log.Println("Finished zipping files")

	return nil
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
