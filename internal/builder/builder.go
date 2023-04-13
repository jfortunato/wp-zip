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

type Operation interface {
	SendFiles(ch chan File) error
}

func initOperations() []Operation {
	return []Operation{
		NewDownloadFilesOperation(),
	}
}

func PackageWP(c Client, outfile io.Writer, pathToPublic string, operations []Operation) error {
	// The resulting archive will consist of the following:
	// 1. All site files, placed into a files/ directory
	// 2. A sql database dump, placed in the root of the archive
	// 3. A JSON file containing some metadata about the site & it's environment, placed in the root of the archive

	// Ensure pathToPublic ends with a slash
	if !strings.HasSuffix(pathToPublic, "/") {
		pathToPublic = pathToPublic + "/"
	}

	// Download/read the wp-config.php file
	configFile, err := downloadSync(c, filepath.Join(pathToPublic, "wp-config.php"))
	if err != nil {
		return fmt.Errorf("error downloading wp-config.php: %s", err)
	}
	_ = configFile

	ch := make(chan File)

	// Create a new zip writer
	zw := zip.NewWriter(outfile)
	defer zw.Close()

	for _, operation := range operations {
		go operation.SendFiles(ch)
	}

	// Write the files into the zip
	for file := range ch {
		err := writeIntoZip(zw, filepath.Join("files", file.Name), file.Body)
		if err != nil {
			return fmt.Errorf("error writing file %s into zip: %s", file.Name, err)
		}
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
