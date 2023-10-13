package database

import (
	"errors"
	"fmt"
	"github.com/jfortunato/wp-zip/internal/emitter"
	"github.com/jfortunato/wp-zip/internal/types"
	"io"
	"regexp"
	"strings"
)

var (
	ErrCouldNotReadWPConfig = errors.New("could not read wp-config.php")
	ErrEmptyContents        = errors.New("empty contents")
	ErrCantFindCredentials  = errors.New("could not find credentials in wp-config.php")
)

// EmitterCredentialsParser is responsible for parsing the database credentials from the wp-config.php file. It uses an Emitter to download the remote file.
type EmitterCredentialsParser struct {
	e          Emitter
	publicPath types.PublicPath
}

// An Emitter is a simpler interface for the emitter.FileEmitter. It is used to download the wp-config.php file.
type Emitter interface {
	EmitSingle(src string, fn emitter.EmitFunc) error
}

// NewEmitterCredentialsParser is a constructor that returns an EmitterCredentialsParser.
func NewEmitterCredentialsParser(e Emitter, publicPath types.PublicPath) *EmitterCredentialsParser {
	return &EmitterCredentialsParser{e, publicPath}
}

// ParseDatabaseCredentials is the main function of the EmitterCredentialsParser. It downloads the wp-config.php file and parses the database credentials from it.
func (p *EmitterCredentialsParser) ParseDatabaseCredentials() (DatabaseCredentials, error) {
	// Download/read the wp-config.php file
	contents, err := p.fetchWPConfigContents()
	if err != nil {
		return DatabaseCredentials{}, fmt.Errorf("%w: %s", ErrCouldNotReadWPConfig, err)
	}

	return parseDatabaseCredentials(contents)
}

// fetchWPConfigContents downloads the wp-config.php file and returns its full contents.
func (p *EmitterCredentialsParser) fetchWPConfigContents() (string, error) {
	var wpConfigFileContents string
	err := p.e.EmitSingle(p.publicPath.String()+"wp-config.php", func(path string, contents io.Reader) {
		wpConfigFileContents = readerToString(contents)
	})
	if err != nil {
		return "", err
	}
	if wpConfigFileContents == "" {
		return "", ErrEmptyContents
	}

	return wpConfigFileContents, nil
}

// parseDatabaseCredentials parses the database credentials from the wp-config.php file.
func parseDatabaseCredentials(contents string) (DatabaseCredentials, error) {
	var fields = map[string]string{"DB_USER": "", "DB_PASSWORD": "", "DB_NAME": "", "DB_HOST": ""}

	for field, _ := range fields {
		value, err := parseWpConfigField(contents, field)
		if err != nil {
			return DatabaseCredentials{}, fmt.Errorf("%w: %s", ErrCantFindCredentials, err)
		}
		fields[field] = value
	}

	return DatabaseCredentials{User: fields["DB_USER"], Pass: fields["DB_PASSWORD"], Name: fields["DB_NAME"], Host: fields["DB_HOST"]}, nil
}

// parseWpConfigField parses a single field from the wp-config.php file.
func parseWpConfigField(contents, field string) (string, error) {
	re := regexp.MustCompile(`define\( *['"]` + field + `['"], *['"](.*)['"] *\);`)
	matches := re.FindStringSubmatch(contents)
	if len(matches) != 2 {
		return "", fmt.Errorf("could not parse %s from wp-config.php", field)
	}
	return matches[1], nil
}

func readerToString(r io.Reader) string {
	buf := new(strings.Builder)
	_, err := io.Copy(buf, r)
	if err != nil {
		panic(err)
	}
	return buf.String()
}
