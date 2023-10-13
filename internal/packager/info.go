package packager

import (
	"errors"
	"fmt"
	"github.com/jfortunato/wp-zip/internal/database"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"github.com/jfortunato/wp-zip/internal/types"
	"io"
	"strings"
)

// This is the SQL statement used to select the site url from the database.
const SELECT_SITE_URL_STMT = "SELECT option_value FROM wp_options WHERE option_name = 'siteurl';"

var ErrCannotParseCredentials = errors.New("error parsing database credentials")

// SiteInfo contains all the information needed to package a WordPress site.
type SiteInfo struct {
	siteUrl       types.SiteUrl
	publicPath    types.PublicPath
	dbCredentials database.DatabaseCredentials
}

type CredentialsParser interface {
	ParseDatabaseCredentials() (database.DatabaseCredentials, error)
}

// DetermineSiteInfo determines the site info needed to package a WordPress site. Some of the information is determined at runtime, such as the database credentials.
func DetermineSiteInfo(siteUrl types.SiteUrl, publicPath types.PublicPath, parser CredentialsParser, runner sftp.RemoteCommandRunner, prompter Prompter) (SiteInfo, error) {
	// We need to determine the database credentials at runtime
	credentials, err := parser.ParseDatabaseCredentials()
	if err != nil {
		return SiteInfo{}, ErrCannotParseCredentials
	}

	// If the siteUrl is empty, we need to determine it at runtime
	if siteUrl == "" {
		siteUrl, err = determineSiteUrl(credentials, runner, prompter)
		if err != nil {
			return SiteInfo{}, err
		}
	}

	// If the publicPath is empty, we need to determine it at runtime
	if publicPath == "" {
		return SiteInfo{}, errors.New("publicPath cannot be empty")
	}

	return SiteInfo{
		siteUrl:       siteUrl,
		publicPath:    publicPath,
		dbCredentials: credentials,
	}, nil
}

func determineSiteUrl(credentials database.DatabaseCredentials, runner sftp.RemoteCommandRunner, prompter Prompter) (types.SiteUrl, error) {
	cmd := fmt.Sprintf(`mysql %s --skip-column-names --silent -e "%s"`, database.MysqlCliCredentials(credentials), SELECT_SITE_URL_STMT)

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
