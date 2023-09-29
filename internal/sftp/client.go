package sftp

import (
	_sftp "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"os"
)

type SSHCredentials struct {
	User string
	Pass string
	Host string
	Port string
}

// RemoteCommandRunner is an interface that allows us to check if the remote server can run a command, and then run it. An object may choose to use this interface instead of a full Client if it only needs to run commands.
type RemoteCommandRunner interface {
	CanRunRemoteCommand(command string) bool
	RunRemoteCommand(command string) (io.Reader, error)
}

// FileUploadDeleter is an interface that allows us to upload, delete, and create directories on the remote server. An object may choose to use this interface instead of a full Client if it only needs to upload/delete files.
type FileUploadDeleter interface {
	Upload(r io.Reader, dst string) error
	Delete(dst string) error
	Mkdir(dst string) error
}

// FileEmitter is an interface that allows us to download files from the remote server. An object may choose to use this interface instead of a full Client if it only needs to download files.
type RemoteFileReader interface {
	ReadDir(path string) ([]os.FileInfo, error)
	Open(path string) (*_sftp.File, error)
	NewSession() (*ssh.Session, error)
}

// A Client is the full interface for interacting with the remote server. It combines the interfaces above.
type Client interface {
	RemoteCommandRunner
	FileUploadDeleter
	RemoteFileReader
}

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

func (c *ClientWrapper) RunRemoteCommand(command string) (io.Reader, error) {
	// We'll pipe the remote output directly into the reader
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		sess, err := c.conn.NewSession()
		if err != nil {
			log.Printf("failed to create session: %s", err)
		}
		sess.Stdout = writer
		defer sess.Close()

		if err := sess.Run(command); err != nil {
			log.Printf("failed to run command: %s", err)
		}
	}()

	return reader, nil
}

func (c *ClientWrapper) Upload(r io.Reader, dst string) error {
	w, err := c.wrapper.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()

	// Copy the contents
	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}

	return nil
}

func (c *ClientWrapper) Delete(dst string) error {
	err := c.wrapper.Remove(dst)
	if err != nil {
		return err
	}

	return nil
}

func (c *ClientWrapper) Mkdir(dst string) error {
	err := c.wrapper.Mkdir(dst)
	if err != nil {
		return err
	}

	return nil
}

func (c *ClientWrapper) Close() error {
	defer c.wrapper.Close()
	return c.conn.Close()
}
