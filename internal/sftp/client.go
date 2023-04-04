package sftp

import (
	"archive/tar"
	"bytes"
	"errors"
	_sftp "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
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

func (c *ClientWrapper) ExportDatabaseToFile(dbUser, dbPass, dbName string, filename string) error {
	if !c.canRunRemoteCommand("mysqldump --version") {
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
	if c.canRunRemoteCommand("tar --version") {
		return c.downloadDirectoryWithTar(remoteDirectory, localDirectory)
	} else {
		return c.downloadDirectoryWithSftp(remoteDirectory, localDirectory)
	}
}

func (c *ClientWrapper) canRunRemoteCommand(cmd string) bool {
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
	defer reader.Close()
	defer writer.Close()

	sess, err := c.conn.NewSession()
	if err != nil {
		return err
	}
	sess.Stdout = writer
	defer sess.Close()

	go func() {
		err := sess.Run("tar -C " + remoteDirectory + " -cf - .")
		if err != nil {
			log.Println(err)
		}
	}()

	tr := tar.NewReader(reader)
	err = untarToDirectory(localDirectory, tr)
	if err != nil {
		return err
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

func (c *ClientWrapper) downloadFile(remoteFile string, localFile string) error {
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

func (c *ClientWrapper) Close() error {
	defer c.wrapper.Close()
	return c.conn.Close()
}
