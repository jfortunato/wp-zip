package builder

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
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
	SendFiles() (<-chan File, error)
}

func initOperations(c Client, pathToPublic string) []Operation {
	downloadFilesOperation, err := NewDownloadFilesOperation(c, pathToPublic)
	if err != nil {
		panic(err)
	}

	return []Operation{
		downloadFilesOperation,
	}
}

func PackageWP(c Client, outfile io.Writer, pathToPublic string, operations []Operation) error {
	// The resulting archive will consist of the following:
	// 1. All site files, placed into a files/ directory
	// 2. A sql database dump, placed in the root of the archive
	// 3. A JSON file containing some metadata about the site & it's environment, placed in the root of the archive

	if len(operations) == 0 {
		return &ErrNoOperations{}
	}

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

	// Create a new zip writer
	zw := zip.NewWriter(outfile)
	defer zw.Close()

	var ch []<-chan File

	for _, operation := range operations {
		channel, _ := operation.SendFiles()
		ch = append(ch, channel)
	}

	// Write the files into the zip
	for file := range merge(ch...) {
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

func merge(cs ...<-chan File) <-chan File {
	var wg sync.WaitGroup
	out := make(chan File)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan File) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
