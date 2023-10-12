package packager

import (
	"errors"
	"github.com/jfortunato/wp-zip/internal/database"
	"github.com/jfortunato/wp-zip/internal/types"
)

var ErrCannotParseCredentials = errors.New("error parsing database credentials")

// SiteInfo contains all the information needed to package a WordPress site.
type SiteInfo struct {
	siteUrl       types.Domain
	publicPath    types.PublicPath
	dbCredentials database.DatabaseCredentials
}

type CredentialsParser interface {
	ParseDatabaseCredentials() (database.DatabaseCredentials, error)
}

// DetermineSiteInfo determines the site info needed to package a WordPress site. Some of the information is determined at runtime, such as the database credentials.
func DetermineSiteInfo(siteUrl types.Domain, publicPath types.PublicPath, parser CredentialsParser) (SiteInfo, error) {
	// If the siteUrl is empty, we need to determine it at runtime
	if siteUrl == "" {
		return SiteInfo{}, errors.New("siteUrl cannot be empty")
	}

	// If the publicPath is empty, we need to determine it at runtime
	if publicPath == "" {
		return SiteInfo{}, errors.New("publicPath cannot be empty")
	}

	// We need to determine the database credentials at runtime
	credentials, err := parser.ParseDatabaseCredentials()
	if err != nil {
		return SiteInfo{}, ErrCannotParseCredentials
	}

	return SiteInfo{
		siteUrl:       siteUrl,
		publicPath:    publicPath,
		dbCredentials: credentials,
	}, nil
}
