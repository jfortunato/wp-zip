package types

import (
	"fmt"
	"net/url"
	"strings"
)

// SiteUrl represents a URL to a WordPress site. We use this type to consistently ensure that the URL is valid.
type SiteUrl string

// NewSiteUrl is a constructor that returns a SiteUrl. It returns an error if the URL is invalid.
func NewSiteUrl(input string) (SiteUrl, error) {
	r, err := url.Parse(input)
	if err != nil {
		return "", err
	}

	if r.Scheme == "" || r.Host == "" {
		return "", fmt.Errorf("invalid url")
	}

	return SiteUrl(fmt.Sprintf("%s://%s", r.Scheme, r.Host)), nil
}

// Domain returns the domain of the SiteUrl without the protocol. For example, if the SiteUrl is https://example.com, this method will return example.com.
func (u SiteUrl) Domain() string {
	// Remove the protocol
	s := strings.TrimPrefix(string(u), "https://")
	return strings.TrimPrefix(s, "http://")
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
