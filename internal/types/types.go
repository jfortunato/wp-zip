package types

import (
	"fmt"
	"net/url"
	"strings"
)

// SiteUrl represents a URL to a WordPress site. We use this type to consistently ensure that the URL is valid.
type SiteUrl struct {
	protocol string
	domain   string
}

// NewSiteUrl is a constructor that returns a SiteUrl. It returns an error if the URL is invalid.
func NewSiteUrl(input string) (SiteUrl, error) {
	r, err := url.Parse(input)
	if err != nil {
		return SiteUrl{}, err
	}

	if r.Scheme == "" || r.Host == "" {
		return SiteUrl{}, fmt.Errorf("invalid url")
	}

	return SiteUrl{r.Scheme, r.Host}, nil
}

// Intended as a stopgap until we can refactor the code to use SiteUrl instead of Domain
func (s *SiteUrl) AsDomain() Domain {
	return Domain(s.domain)
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
