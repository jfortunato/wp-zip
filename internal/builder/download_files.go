package builder

type downloadFilesOperation struct {
}

func NewDownloadFilesOperation() Operation {
	return &downloadFilesOperation{}
}

func (o *downloadFilesOperation) SendFiles(ch chan File) error {
	return nil
}
