package packager

import (
	"errors"
	"fmt"
	"github.com/jfortunato/wp-zip/internal/database"
	"github.com/jfortunato/wp-zip/internal/emitter"
	"github.com/jfortunato/wp-zip/internal/operations"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"github.com/jfortunato/wp-zip/internal/types"
	"io"
	"os"
)

var (
	ErrCannotCreateClient      = errors.New("cannot create client")
	ErrCannotDetermineSiteInfo = errors.New("cannot determine site info")
	ErrCannotBuildOperations   = errors.New("cannot build operations")
	ErrCannotCreateZipFile     = errors.New("cannot create zip file")
	ErrCannotRunOperations     = errors.New("cannot run operations")
)

// OperationsBuilder builds all the operations needed to package a WordPress site. They will be run by the OperationsRunner.
type OperationsBuilder interface {
	Build(info SiteInfo) ([]operations.Operation, error)
}

// OperationsRunner runs all the operations needed to package a WordPress site. The operations are first built by the OperationsBuilder. Typically, the writer would be a zip file to output to.
type OperationsRunner interface {
	Run(operations []operations.Operation, writer io.Writer) error
}

// Packager is the orchestrator of the packaging process. It uses an OperationsBuilder to build the operations, and an OperationsRunner to run them. It is also responsible for creating the zip file to output to.
type Packager struct {
	b OperationsBuilder
	r OperationsRunner
	i SiteInfo
}

// NewPackager is the constructor for Packager. It will create the default implementations of OperationsBuilder and OperationsRunner.
func NewPackager(sshCredentials sftp.SSHCredentials, publicUrl types.Domain, publicPath types.PublicPath) (*Packager, error) {
	client, err := sftp.NewClient(sshCredentials)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCannotCreateClient, err)
	}

	e := emitter.NewFileEmitter(client)

	info, err := DetermineSiteInfo(publicUrl, publicPath, database.NewEmitterCredentialsParser(e, publicPath), client, &RuntimePrompter{})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCannotDetermineSiteInfo, err)
	}

	builder := &Builder{
		c: client,
		e: e,
		g: &operations.BasicHttpGetter{},
	}

	return &Packager{builder, &Runner{}, info}, nil
}

// PackageWP packages a WordPress site. It will build the operations, run them, and output the zip file.
func (p *Packager) PackageWP(outputFilename string) error {
	// The resulting archive will consist of the following:
	// 1. All site files, placed into a files/ directory
	// 2. A sql database dump, placed in the root of the archive
	// 3. A JSON file containing some metadata about the site & it's environment, placed in the root of the archive
	ops, err := p.b.Build(p.i)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCannotBuildOperations, err)
	}

	zipFile, err := os.Create(outputFilename)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCannotCreateZipFile, err)
	}
	defer zipFile.Close()

	err = p.r.Run(ops, zipFile)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCannotRunOperations, err)
	}

	return nil
}
