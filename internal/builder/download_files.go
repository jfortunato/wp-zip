package builder

import (
	"errors"
	"strings"
)

type downloadFilesOperation struct {
	c            Client
	pathToPublic string
}

func NewDownloadFilesOperation(c Client, pathToPublic string) (Operation, error) {
	// Assert pathToPublic ends with a slash
	if !strings.HasSuffix(pathToPublic, "/") {
		return nil, errors.New("pathToPublic must end with a slash")
	}

	return &downloadFilesOperation{c, pathToPublic}, nil
}

func (o *downloadFilesOperation) SendFiles() (<-chan File, error) {
	ch := make(chan File)

	go func() {
		o.c.Download(o.pathToPublic, ch)
	}()

	return ch, nil
}
