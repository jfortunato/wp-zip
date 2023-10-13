package packager

import (
	"errors"
	"github.com/jfortunato/wp-zip/internal/database"
	"github.com/jfortunato/wp-zip/internal/types"
	"io"
	"strings"
	"testing"
)

func TestDetermineSiteInfo(t *testing.T) {
	t.Run("it should return the site info", func(t *testing.T) {
		got, err := DetermineSiteInfo("localhost", "public", &CredentialsParserStub{}, &MockCommandRunner{}, &PrompterSpy{})

		if err != nil {
			t.Errorf("got error %v; want nil", err)
		}

		// Assert that we got the site info we expect
		want := SiteInfo{"localhost", "public", database.DatabaseCredentials{"user", "pass", "db", "localhost"}}
		if got != want {
			t.Errorf("got site info %v; want %v", got, want)
		}
	})

	t.Run("it should return an error if the credentials parser fails", func(t *testing.T) {
		parser := &CredentialsParserStub{errorStub: errors.New("error")}

		_, err := DetermineSiteInfo("localhost", "public", parser, &MockCommandRunner{}, &PrompterSpy{})

		// Assert that we got the error we expect
		if !errors.Is(err, ErrCannotParseCredentials) {
			t.Errorf("got error %v; want ErrCannotParseCredentials", err)
		}
	})

	t.Run("it should determine the site url at runtime if not given", func(t *testing.T) {
		cmd := `mysql --user='user' --password='pass' --host=localhost db --skip-column-names --silent -e "SELECT option_value FROM wp_options WHERE option_name = 'siteurl';"`

		var tests = []struct {
			name        string
			stubbedCmds map[string]string
			promptCalls int
			wantSiteUrl types.Domain
		}{
			{
				"via database select",
				map[string]string{
					cmd: "https://example.com/\n",
				},
				0,
				"example.com",
			},
			{
				"via prompter if cmd cannot be run",
				map[string]string{},
				1,
				"prompted-localhost",
			},
			{
				"via prompter if cmd can be run but returns empty string",
				map[string]string{
					cmd: "",
				},
				1,
				"prompted-localhost",
			},
			{
				"via prompter if cmd can be run but returns invalid url",
				map[string]string{
					cmd: "invalid-url",
				},
				1,
				"prompted-localhost",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				prompter := &PrompterSpy{}

				got, err := DetermineSiteInfo("", "public", &CredentialsParserStub{}, &MockCommandRunner{tt.stubbedCmds}, prompter)

				if err != nil {
					t.Errorf("got error %v; want nil", err)
				}

				want := SiteInfo{tt.wantSiteUrl, "public", database.DatabaseCredentials{"user", "pass", "db", "localhost"}}
				if got != want {
					t.Errorf("got site info %v; want %v", got, want)
				}

				// Assert that the prompter was called the correct number of times
				if prompter.calls != tt.promptCalls {
					t.Errorf("got %d prompt calls; want %d", prompter.calls, tt.promptCalls)
				}
			})
		}
	})
}

type CredentialsParserStub struct {
	errorStub error
}

func (p *CredentialsParserStub) ParseDatabaseCredentials() (database.DatabaseCredentials, error) {
	return database.DatabaseCredentials{"user", "pass", "db", "localhost"}, p.errorStub
}

type PrompterSpy struct {
	calls int
}

func (p *PrompterSpy) Prompt(question string) string {
	p.calls++
	return "prompted-localhost"
}

type MockCommandRunner struct {
	commandsThatExist map[string]string
}

func (m *MockCommandRunner) CanRunRemoteCommand(command string) bool {
	_, ok := m.commandsThatExist[command]
	return ok
}

func (m *MockCommandRunner) RunRemoteCommand(command string) (io.Reader, error) {
	return strings.NewReader(m.commandsThatExist[command]), nil
}
