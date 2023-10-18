package packager

import (
	"errors"
	"fmt"
	"github.com/jfortunato/wp-zip/internal/database"
	"github.com/jfortunato/wp-zip/internal/parser"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"github.com/jfortunato/wp-zip/internal/types"
	"io"
	"strings"
)

// This is the SQL statement used to select the site url from the database.
const SELECT_SITE_URL_STMT = "SELECT option_value FROM %s WHERE option_name = 'siteurl';"

var ErrCannotParseWPConfig = errors.New("error parsing wp-config.php file")

// SiteInfo contains all the information needed to package a WordPress site.
type SiteInfo struct {
	siteUrl       types.SiteUrl
	publicPath    types.PublicPath
	dbCredentials database.DatabaseCredentials
}

type WPConfigParser interface {
	ParseWPConfig(publicPath types.PublicPath) (parser.WPConfigFields, error)
}

// DetermineSiteInfo determines the site info needed to package a WordPress site. Some of the information is determined at runtime, such as the database credentials.
func DetermineSiteInfo(siteUrl types.SiteUrl, publicPath types.PublicPath, parser WPConfigParser, runner sftp.RemoteCommandRunner, prompter Prompter) (SiteInfo, error) {
	var err error

	// If the publicPath is empty, we need to determine it at runtime
	if publicPath == "" {
		publicPath, err = determinePublicPath(runner, prompter)
		if err != nil {
			return SiteInfo{}, err
		}
	}

	// We need to determine the database credentials & table prefix at runtime
	fields, err := parser.ParseWPConfig(publicPath)
	if err != nil {
		return SiteInfo{}, fmt.Errorf("%w: %s", ErrCannotParseWPConfig, err)
	}

	// If the siteUrl is empty, we need to determine it at runtime
	if siteUrl == "" {
		siteUrl, err = determineSiteUrl(fields, runner, prompter)
		if err != nil {
			return SiteInfo{}, err
		}
	}

	return SiteInfo{
		siteUrl:       siteUrl,
		publicPath:    publicPath,
		dbCredentials: fields.Credentials,
	}, nil
}

func determineSiteUrl(fields parser.WPConfigFields, runner sftp.RemoteCommandRunner, prompter Prompter) (types.SiteUrl, error) {
	stmt := fmt.Sprintf(SELECT_SITE_URL_STMT, fields.Prefix+"options")
	cmd := fmt.Sprintf(`mysql %s --skip-column-names --silent -e "%s"`, database.MysqlCliCredentials(fields.Credentials), stmt)

	var siteUrl types.SiteUrl

	// First we'll try to automatically get the siteurl from the database.
	if runner.CanRunRemoteCommand(cmd) {
		siteUrl = queryForSiteUrl(runner, cmd)
	}

	// If we don't have a siteUrl at this point, we need to prompt for it
	if siteUrl == "" {
		var err error
		siteUrl, err = promptForSiteUrl(prompter)
		if err != nil {
			return "", err
		}
	}

	return siteUrl, nil
}

func queryForSiteUrl(runner sftp.RemoteCommandRunner, cmd string) types.SiteUrl {
	output, err := runner.RunRemoteCommand(cmd)
	if err != nil {
		return ""
	}
	// Convert the output to a string & trim whitespace
	b, _ := io.ReadAll(output)
	str := strings.TrimSpace(string(b))
	u, _ := types.NewSiteUrl(str)
	return u
}

func promptForSiteUrl(prompter Prompter) (types.SiteUrl, error) {
	response := prompter.Prompt("What is the site url?")
	u, err := types.NewSiteUrl(response)
	if err != nil {
		return "", err
	}
	return u, nil
}

func determinePublicPath(runner sftp.RemoteCommandRunner, prompter Prompter) (types.PublicPath, error) {
	cmd := `find -L . -type f -name 'wp-config.php'`

	var publicPath types.PublicPath

	if runner.CanRunRemoteCommand(cmd) {
		output, err := runner.RunRemoteCommand(cmd)
		if err != nil {
			return "", err
		}
		// If there are multiple results, we don't want to guess which one is the correct one, so we'll prompt for the public path
		// We'll determine that there are multiple results by counting the number of newlines in the output
		b, _ := io.ReadAll(output)
		totalNewLines := strings.Count(string(b), "\n")
		if totalNewLines <= 1 {
			str := strings.TrimSpace(string(b))
			publicPath = types.PublicPath(strings.TrimSuffix(str, "/wp-config.php"))
		}
	}

	if publicPath == "" {
		var err error
		publicPath, err = promptForPublicPath(prompter)
		if err != nil {
			return "", err
		}
	}

	return publicPath, nil
}

func promptForPublicPath(prompter Prompter) (types.PublicPath, error) {
	response := prompter.Prompt("What is the public path?")
	if response == "" {
		return "", errors.New("public path cannot be empty")
	}
	return types.PublicPath(response), nil
}
