package builder

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type ErrNoOperations struct{}

func (e *ErrNoOperations) Error() string {
	return "no operations to run"
}

type File struct {
	Name string
	Body io.Reader
}

type Client interface {
	Upload(src io.Reader, dst string) error
	Download(src string, ch chan File) error
	Run(cmd string) ([]byte, error)
}

type SendFilesFunc func(file File) error

type Operation interface {
	SendFiles(fn SendFilesFunc) error
}

// PublicPath represents the path to the public directory of a WordPress site. We use this
// type to consistently ensure that the path ends with a slash when used as a string.
type PublicPath string

func (p PublicPath) String() string {
	// If p doesn't end in a slash, add one
	if !strings.HasSuffix(string(p), "/") {
		return string(p) + "/"
	}

	return string(p)
}

//func initOperations(c Client, pathToPublic PublicPath) []Operation {
//	downloadFilesOperation, err := NewDownloadFilesOperation(c, pathToPublic)
//	if err != nil {
//		panic(err)
//	}
//
//	return []Operation{
//		downloadFilesOperation,
//	}
//}

func PackageWP(c Client, outfile io.Writer, pathToPublic PublicPath, operations []Operation) error {
	// The resulting archive will consist of the following:
	// 1. All site files, placed into a files/ directory
	// 2. A sql database dump, placed in the root of the archive
	// 3. A JSON file containing some metadata about the site & it's environment, placed in the root of the archive

	if len(operations) == 0 {
		return &ErrNoOperations{}
	}

	// Download/read the wp-config.php file
	configFile, err := downloadSync(c, filepath.Join(string(pathToPublic), "wp-config.php"))
	if err != nil {
		return fmt.Errorf("error downloading wp-config.php: %s", err)
	}
	_ = configFile

	// Create a new zip writer
	zw := zip.NewWriter(outfile)
	defer zw.Close()

	for _, operation := range operations {
		operation.SendFiles(func(file File) error {
			// Write the files into the zip
			err := writeIntoZip(zw, filepath.Join("files", file.Name), file.Body)
			if err != nil {
				return fmt.Errorf("error writing file %s into zip: %s", file.Name, err)
			}
			return nil
		})
	}

	return nil
}

func downloadSync(c Client, src string) (File, error) {
	ch := make(chan File, 1)

	err := c.Download(src, ch)
	if err != nil {
		return File{}, err
	}

	return <-ch, nil
}

func writeIntoZip(zw *zip.Writer, filename string, contents io.Reader) error {
	f, err := zw.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file %s in zip: %s", filename, err)
	}
	_, err = io.Copy(f, contents)
	if err != nil {
		return fmt.Errorf("error copying file %s into zip: %s", filename, err)
	}
	return nil
}
