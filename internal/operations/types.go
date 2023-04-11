package operations

import "archive/zip"

type Operation interface {
	WriteIntoZip(zw *zip.Writer) error
}
