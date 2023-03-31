package sftp

import (
	"bytes"
	_sftp "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
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

func (c *ClientWrapper) Close() error {
	defer c.wrapper.Close()
	return c.conn.Close()
}
