package sftp

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	_sftp "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SSHCredentials struct {
	User string
	Pass string
	Host string
	Port string
}

// Client Our Client interface is a wrapper around the pkg/sftp Client
// so that we can require it to implement only a subset of the
// methods that we actually need.
//type Client interface {
//	ReadFileToString(path string) (string, error)
//	Close() error
//}

// ClientWrapper Our ClientWrapper is a wrapper around the pkg/sftp Client
// that only exposes the methods that we actually need.
type ClientWrapper struct {
	wrapper *_sftp.Client
	conn    *ssh.Client
}

func NewClient(credentials SSHCredentials) (*ClientWrapper, error) {
	// Set up the config
	config := &ssh.ClientConfig{
		User:            credentials.User,
		Auth:            []ssh.AuthMethod{ssh.Password(credentials.Pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Set up the connection
	conn, err := ssh.Dial("tcp", net.JoinHostPort(credentials.Host, credentials.Port), config)
	if err != nil {
		return nil, err
	}
	// Don't close here, our ClientWrapper is responsible for
	// closing both the sftp client and the ssh connection.

	client, err := _sftp.NewClient(conn)
	if err != nil {
		// If we couldn't create the sftp client, close the ssh connection
		conn.Close()
		return nil, err
	}
	// Don't close here, our ClientWrapper is responsible for
	// closing both the sftp client and the ssh connection.

	return &ClientWrapper{client, conn}, nil
}

func (c *ClientWrapper) ReadDir(path string) ([]os.FileInfo, error) {
	return c.wrapper.ReadDir(path)
}

func (c *ClientWrapper) Open(path string) (*_sftp.File, error) {
	return c.wrapper.Open(path)
}

func (c *ClientWrapper) NewSession() (*ssh.Session, error) {
	return c.conn.NewSession()
}

func (c *ClientWrapper) ReadFileToString(path string) (string, error) {
	file, err := c.wrapper.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var b bytes.Buffer
	_, err = io.Copy(&b, file)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func (c *ClientWrapper) WriteZip(directory, zipFilename string) error {
	// Create the zip file
	zipFile, err := os.Create(zipFilename)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %s", err)
	}
	defer zipFile.Close()

	// Create the zip writer
	w := zip.NewWriter(zipFile)
	defer w.Close()

	// Walk the directory
	return filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
		// Convert the path to a relative path
		relativePath, err := filepath.Rel(directory, path)

		// Skip the first entry, which is the directory itself
		if path == directory {
			return nil
		}

		// The writer's Create method will create a directory if the path ends with a slash
		if d.IsDir() {
			relativePath += "/"
		}

		writer, err := w.Create(relativePath)
		if err != nil {
			return fmt.Errorf("failed to create zip entry: %w", err)
		}

		// We don't need to copy any contents if it's a directory
		if d.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer f.Close()

		_, err = io.Copy(writer, f)
		if err != nil {
			return fmt.Errorf("failed to copy file to zip: %w", err)
		}

		return nil
	})
}

func (c *ClientWrapper) GenerateJsonFile(dbUser, dbPass, dbName, publicUrl, publicPath, filename string) error {
	contents := fmt.Sprintf(`<?php

// Connect to mysql and get the mysql version
$link = mysqli_connect('localhost', '%s', '%s', '%s');
$mysqlVersion = mysqli_get_server_info($link);
mysqli_close($link);

// Get the server name and version
preg_match('/^(apache|nginx)\/(\d+\.\d+\.\d+).*/', strtolower($_SERVER['SERVER_SOFTWARE']), $matches);
$serverJson = isset($matches[1], $matches[2]) ? [ $matches[1] => [ 'name' => $matches[1], 'version' => $matches[2] ] ] : '';

// Get the current WordPress version by reading the wp-includes/version.php file
$wpVersionFile = file_get_contents(__DIR__ . DIRECTORY_SEPARATOR . 'wp-includes' . DIRECTORY_SEPARATOR .  'version.php');
preg_match('/\$wp_version = \'(.*)\';/', $wpVersionFile, $matches);
$wpVersion = isset($matches[1]) ? $matches[1] : '';

header('Content-Type: application/json');
echo json_encode(array_merge_recursive([
    'name' => 'Migrated Site',
    'domain' => '%s',
    'path' => '%s',
    'wpVersion' => $wpVersion,
    'services' => [
        'php' => [
            'name' => 'php',
            'version' => PHP_VERSION,
        ],
        'mysql' => [
            'name' => 'mysql',
            'version' => $mysqlVersion,
        ],
    ],
], ['services' => $serverJson]));
`, dbUser, dbPass, dbName, publicUrl, publicPath)

	// Generate a random filename
	uploadFilename := "wp-zip-" + randSeq(10) + ".php"

	// Upload the contents to the server
	// Assuming a Unix server with a "/" path separator
	err := c.uploadContentsToFile(strings.NewReader(contents), publicPath+"/"+uploadFilename)
	if err != nil {
		return err
	}
	defer c.deleteFile(publicPath + "/" + uploadFilename)

	// Make a request to the file to get the JSON
	body, err := readHttpResonseToString(publicUrl + "/" + uploadFilename)
	if err != nil {
		return err
	}

	// Write the body to a json string
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, strings.NewReader(body))
	if err != nil {
		return err
	}

	return nil
}

func (c *ClientWrapper) ExportDatabaseToFile(dbUser, dbPass, dbName string, filename string) error {
	if !c.CanRunRemoteCommand("mysqldump --version") {
		return errors.New("mysqldump not found on remote server")
	}

	sess, err := c.conn.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	sess.Stdout = file

	err = sess.Run("mysqldump --no-tablespaces -u " + dbUser + " -p" + dbPass + " " + dbName)
	if err != nil {
		return err
	}

	return nil
}

func (c *ClientWrapper) DownloadDirectory(remoteDirectory string, localDirectory string) error {
	// First see if we can use the tar client, then fall back to the sftp client
	if c.CanRunRemoteCommand("tar --version") {
		return c.downloadDirectoryWithTar(remoteDirectory, localDirectory)
	} else {
		return c.downloadDirectoryWithSftp(remoteDirectory, localDirectory)
	}
}

func (c *ClientWrapper) CanRunRemoteCommand(cmd string) bool {
	sess, err := c.conn.NewSession()
	if err != nil {
		return false
	}
	defer sess.Close()

	// Check that the remote session can successfully run the command
	err = sess.Run(cmd)
	if err != nil {
		return false
	}

	return true
}

func (c *ClientWrapper) downloadDirectoryWithTar(remoteDirectory string, localDirectory string) error {
	// We'll pipe the remote tar output directly into the tar reader
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		sess, err := c.conn.NewSession()
		if err != nil {
			log.Fatalln("failed to create session: %w", err)
		}
		sess.Stdout = writer
		defer sess.Close()

		if err := sess.Run("tar -C " + remoteDirectory + " -cf - ."); err != nil {
			log.Fatalln("failed to run tar: %w", err)
		}
	}()

	tr := tar.NewReader(reader)
	if err := untarToDirectory(localDirectory, tr); err != nil {
		fmt.Errorf("failed to untar: %w", err)
	}

	return nil
}

func untarToDirectory(localDirectory string, tr *tar.Reader) error {
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Remove any trailing slashes
		header.Name = strings.TrimSuffix(header.Name, "/")
		// Split the path into its components
		paths := strings.Split(header.Name, "/")
		// Ignore the first path (name should be something like "./foo, so disregard the ".)
		paths = paths[1:]
		// Don't do anything for the top level directory
		if len(paths) == 0 {
			continue
		}
		// Prefix the path with the local directory
		paths = append([]string{localDirectory}, paths...)

		targetPath := filepath.Join(paths...)

		// Check if the file is a directory or a regular file
		if header.FileInfo().IsDir() {
			// Create the directory
			err = os.Mkdir(targetPath, 0755)
			if err != nil {
				return err
			}
		} else {
			// Create the file
			f, _ := os.Create(targetPath)
			_, err = io.Copy(f, tr)
			if err != nil {
				return err
			}
			f.Close()
		}
	}

	return nil
}

func (c *ClientWrapper) downloadDirectoryWithSftp(remoteDirectory string, localDirectory string) error {
	// Get the list of files in the remote directory
	remoteFiles, err := c.wrapper.ReadDir(remoteDirectory)
	if err != nil {
		return err
	}
	for _, remoteFile := range remoteFiles {
		remoteFilepath := filepath.Join(remoteDirectory, remoteFile.Name())
		localFilepath := filepath.Join(localDirectory, remoteFile.Name())

		// If the file is a directory, recursively download it
		if remoteFile.IsDir() {
			// Create the local directory
			err = os.Mkdir(localFilepath, 0755)
			if err != nil {
				return err
			}

			// Recursively download the directory
			err = c.DownloadDirectory(remoteFilepath, localFilepath)
			if err != nil {
				return err
			}
		} else {
			// Otherwise, download the file
			err = c.downloadFile(remoteFilepath, localFilepath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *ClientWrapper) downloadFile(remoteFile, localFile string) error {
	r, err := c.wrapper.Open(remoteFile)
	if err != nil {
		return err
	}
	defer r.Close()

	// Create the local file
	w, err := os.Create(localFile)
	if err != nil {
		return err
	}
	defer w.Close()

	// Copy the file
	log.Println("Downloading " + remoteFile)
	_, err = w.ReadFrom(r)
	if err != nil {
		return err
	}

	return nil
}

func (c *ClientWrapper) deleteFile(remoteFile string) error {
	err := c.wrapper.Remove(remoteFile)
	if err != nil {
		return err
	}

	return nil
}

func (c *ClientWrapper) uploadContentsToFile(contents io.Reader, remoteFile string) error {
	w, err := c.wrapper.Create(remoteFile)
	if err != nil {
		return err
	}
	defer w.Close()

	// Copy the contents
	log.Println("Uploading contents")
	_, err = io.Copy(w, contents)
	if err != nil {
		return err
	}

	return nil
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	rand.Seed(time.Now().UnixNano())
	//r := rand.New(rand.NewSource(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func readHttpResonseToString(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (c *ClientWrapper) Close() error {
	defer c.wrapper.Close()
	return c.conn.Close()
}
