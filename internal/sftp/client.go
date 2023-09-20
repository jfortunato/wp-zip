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

func (c *ClientWrapper) Close() error {
	defer c.wrapper.Close()
	return c.conn.Close()
}
