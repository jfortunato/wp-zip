package types

import (
	"fmt"
	"strings"
)

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
