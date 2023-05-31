package builder

type downloadFilesOperation struct {
	c            Client
	pathToPublic PublicPath
}

func NewDownloadFilesOperation(c Client, pathToPublic PublicPath) (Operation, error) {
	return &downloadFilesOperation{c, pathToPublic}, nil
}

func (o *downloadFilesOperation) SendFiles() (<-chan File, error) {
	ch := make(chan File)

	go func() {
		o.c.Download(string(o.pathToPublic), ch)
	}()

	return ch, nil
}
