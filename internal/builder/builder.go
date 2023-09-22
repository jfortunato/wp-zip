package builder

import (
	"archive/zip"
	"fmt"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"io"
	"regexp"
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

type SendFilesFunc func(file File) error

type Operation interface {
	SendFiles(fn SendFilesFunc) error
}

// Domain represents a domain name/host. We can use the AsSecureUrl and AsInsecureUrl methods
// to get the domain as a URL with the appropriate protocol.
type Domain string

func (d Domain) AsSecureUrl() string {
	return fmt.Sprintf("https://%s", d)
}

func (d Domain) AsInsecureUrl() string {
	return fmt.Sprintf("http://%s", d)
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

func initFileEmitter(c *sftp.ClientWrapper) FileEmitter {
	return NewFileEmitter(c, c)
}

type Builder struct {
	publicPath PublicPath
	operations []Operation
}

func (b *Builder) PackageWP(outfile io.Writer) error {
	// The resulting archive will consist of the following:
	// 1. All site files, placed into a files/ directory
	// 2. A sql database dump, placed in the root of the archive
	// 3. A JSON file containing some metadata about the site & it's environment, placed in the root of the archive

	if len(b.operations) == 0 {
		return &ErrNoOperations{}
	}

	// Create a new zip writer
	zw := zip.NewWriter(outfile)
	defer zw.Close()

	for _, operation := range b.operations {
		err := operation.SendFiles(func(file File) error {
			// Write the files into the zip
			err := writeIntoZip(zw, file)
			if err != nil {
				return fmt.Errorf("error writing file %s into zip: %s", file.Name, err)
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("error sending files: %s", err)
		}
	}

	return nil
}

func writeIntoZip(zw *zip.Writer, file File) error {
	f, err := zw.Create(file.Name)
	if err != nil {
		return fmt.Errorf("error creating file %s in zip: %s", file.Name, err)
	}
	_, err = io.Copy(f, file.Body)
	if err != nil {
		return fmt.Errorf("error copying file %s into zip: %s", file.Name, err)
	}
	return nil
}

func readerToString(r io.Reader) string {
	buf := new(strings.Builder)
	_, err := io.Copy(buf, r)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func parseDatabaseCredentials(contents string) (DatabaseCredentials, error) {
	var fields = map[string]string{"DB_USER": "", "DB_PASSWORD": "", "DB_NAME": ""}

	for field, _ := range fields {
		value, err := parseWpConfigField(contents, field)
		if err != nil {
			return DatabaseCredentials{}, err
		}
		fields[field] = value
	}

	return DatabaseCredentials{User: fields["DB_USER"], Pass: fields["DB_PASSWORD"], Name: fields["DB_NAME"]}, nil
}

func parseWpConfigField(contents, field string) (string, error) {
	// TODO: Add tests for multiple different formats for these fields
	re := regexp.MustCompile(`define\( ?['"]` + field + `['"], ['"](.*)['"] ?\);`)
	matches := re.FindStringSubmatch(contents)
	if len(matches) != 2 {
		return "", fmt.Errorf("could not parse %s from wp-config.php", field)
	}
	return matches[1], nil
}
