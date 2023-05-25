package builder

type downloadFilesOperation struct {
	c            Client
	pathToPublic string
}

func NewDownloadFilesOperation(c Client, pathToPublic string) Operation {
	return &downloadFilesOperation{c, pathToPublic}
}

func (o *downloadFilesOperation) SendFiles() (<-chan File, error) {
	ch := make(chan File)

	go func() {
		o.c.Download(o.pathToPublic, ch)
	}()

	return ch, nil
}
