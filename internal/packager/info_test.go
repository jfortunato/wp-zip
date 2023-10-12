package packager

import (
	"errors"
	"github.com/jfortunato/wp-zip/internal/database"
	"testing"
)

func TestDetermineSiteInfo(t *testing.T) {
	t.Run("it should return the site info", func(t *testing.T) {
		parser := &CredentialsParserStub{}

		got, err := DetermineSiteInfo("localhost", "public", parser)

		if err != nil {
			t.Errorf("got error %v; want nil", err)
		}

		// Assert that we got the site info we expect
		want := SiteInfo{"localhost", "public", database.DatabaseCredentials{}}

		if got != want {
			t.Errorf("got site info %v; want %v", got, want)
		}
	})

	t.Run("it should return an error if the credentials parser fails", func(t *testing.T) {
		parser := &CredentialsParserStub{errorStub: errors.New("error")}

		_, err := DetermineSiteInfo("localhost", "public", parser)

		// Assert that we got the error we expect
		if !errors.Is(err, ErrCannotParseCredentials) {
			t.Errorf("got error %v; want ErrCannotParseCredentials", err)
		}
	})
}

type CredentialsParserStub struct {
	errorStub error
}

func (p *CredentialsParserStub) ParseDatabaseCredentials() (database.DatabaseCredentials, error) {
	return database.DatabaseCredentials{}, p.errorStub
}
