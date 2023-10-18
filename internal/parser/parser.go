package parser

import (
	"errors"
	"fmt"
	"github.com/jfortunato/wp-zip/internal/database"
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
	ErrCantFindPrefix       = errors.New("could not find prefix in wp-config.php")
)

// EmitterWPConfigParser is responsible for parsing the required fields from the wp-config.php file. It uses an Emitter to download the remote file.
type EmitterWPConfigParser struct {
	e Emitter
}

// An Emitter is a simpler interface for the emitter.FileEmitter. It is used to download the wp-config.php file.
type Emitter interface {
	EmitSingle(src string, fn emitter.EmitFunc) error
}

// WPConfigFields holds the fields parsed from the wp-config.php file.
type WPConfigFields struct {
	Credentials database.DatabaseCredentials
	Prefix      string
}

// NewEmitterCredentialsParser is a constructor that returns an EmitterWPConfigParser.
func NewEmitterCredentialsParser(e Emitter) *EmitterWPConfigParser {
	return &EmitterWPConfigParser{e}
}

// ParseWPConfig is the main function of the EmitterWPConfigParser. It downloads the wp-config.php file and parses the fields we need
// (database credentials, table prefix) from it.
func (p *EmitterWPConfigParser) ParseWPConfig(publicPath types.PublicPath) (WPConfigFields, error) {
	// Download/read the wp-config.php file
	contents, err := p.fetchWPConfigContents(publicPath)
	if err != nil {
		return WPConfigFields{}, fmt.Errorf("%w: %s", ErrCouldNotReadWPConfig, err)
	}

	credentials, err := parseDatabaseCredentials(contents)
	if err != nil {
		return WPConfigFields{}, fmt.Errorf("%w: %s", ErrCantFindCredentials, err)
	}

	prefix, err := parsePrefix(contents)
	if err != nil {
		return WPConfigFields{}, fmt.Errorf("%w: %s", ErrCantFindPrefix, err)
	}

	return WPConfigFields{Credentials: credentials, Prefix: prefix}, nil
}

// fetchWPConfigContents downloads the wp-config.php file and returns its full contents.
func (p *EmitterWPConfigParser) fetchWPConfigContents(publicPath types.PublicPath) (string, error) {
	var wpConfigFileContents string
	err := p.e.EmitSingle(publicPath.String()+"wp-config.php", func(path string, contents io.Reader) {
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
func parseDatabaseCredentials(contents string) (database.DatabaseCredentials, error) {
	var fields = map[string]string{"DB_USER": "", "DB_PASSWORD": "", "DB_NAME": "", "DB_HOST": ""}

	for field, _ := range fields {
		value, err := parseWpConfigField(contents, field)
		if err != nil {
			return database.DatabaseCredentials{}, fmt.Errorf("%w: %s", ErrCantFindCredentials, err)
		}
		fields[field] = value
	}

	return database.DatabaseCredentials{User: fields["DB_USER"], Pass: fields["DB_PASSWORD"], Name: fields["DB_NAME"], Host: fields["DB_HOST"]}, nil
}

// parsePrefix parses the table name prefix from the wp-config.php file.
func parsePrefix(contents string) (string, error) {
	re := regexp.MustCompile(`\$table_prefix  *= *['"](.*)['"];`)
	matches := re.FindStringSubmatch(contents)
	if len(matches) != 2 {
		return "", fmt.Errorf("could not parse table prefix from wp-config.php")
	}
	return matches[1], nil
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
