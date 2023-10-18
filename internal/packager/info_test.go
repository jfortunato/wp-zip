package packager

import (
	"errors"
	"github.com/jfortunato/wp-zip/internal/database"
	"github.com/jfortunato/wp-zip/internal/parser"
	"github.com/jfortunato/wp-zip/internal/types"
	"io"
	"strings"
	"testing"
)

func TestDetermineSiteInfo(t *testing.T) {
	t.Run("it should return the site info", func(t *testing.T) {
		got, err := DetermineSiteInfo("localhost", "public", newConfigParserStub(), &MockCommandRunner{}, &PrompterSpy{})

		if err != nil {
			t.Errorf("got error %v; want nil", err)
		}

		// Assert that we got the site info we expect
		want := SiteInfo{"localhost", "public", database.DatabaseCredentials{"user", "pass", "db", "localhost"}}
		if got != want {
			t.Errorf("got site info %v; want %v", got, want)
		}
	})

	t.Run("it should return an error if the wp config parser fails", func(t *testing.T) {
		parser := newConfigParserStub()
		parser.errorStub = errors.New("error")

		_, err := DetermineSiteInfo("localhost", "public", parser, &MockCommandRunner{}, &PrompterSpy{})

		// Assert that we got the error we expect
		if !errors.Is(err, ErrCannotParseWPConfig) {
			t.Errorf("got error %v; want ErrCannotParseWPConfig", err)
		}
	})

	t.Run("it should determine the site url at runtime if not given", func(t *testing.T) {
		cmds := map[string]string{
			"default":            `mysql --user='user' --password='pass' --host=localhost db --skip-column-names --silent -e "SELECT option_value FROM wp_options WHERE option_name = 'siteurl';"`,
			"alternative-prefix": `mysql --user='user' --password='pass' --host=localhost db --skip-column-names --silent -e "SELECT option_value FROM xx_options WHERE option_name = 'siteurl';"`,
		}

		var tests = []struct {
			name        string
			stubbedCmds map[string]string
			prefix      string
			promptCalls int
			wantSiteUrl types.SiteUrl
		}{
			{
				"via database select",
				map[string]string{
					cmds["default"]: "https://example.com/\n",
				},
				"wp_",
				0,
				"https://example.com",
			},
			{
				"via prompter if cmd cannot be run",
				map[string]string{},
				"wp_",
				1,
				"http://prompted-localhost",
			},
			{
				"via prompter if cmd can be run but returns empty string",
				map[string]string{
					cmds["default"]: "",
				},
				"wp_",
				1,
				"http://prompted-localhost",
			},
			{
				"via prompter if cmd can be run but returns invalid url",
				map[string]string{
					cmds["default"]: "invalid-url",
				},
				"wp_",
				1,
				"http://prompted-localhost",
			},
			{
				"via database select - alternative prefix",
				map[string]string{
					cmds["alternative-prefix"]: "https://example.com/\n",
				},
				"xx_",
				0,
				"https://example.com",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				prompter := &PrompterSpy{}
				parser := newConfigParserStub()
				parser.fieldsStub.Prefix = tt.prefix

				got, err := DetermineSiteInfo("", "public", parser, &MockCommandRunner{tt.stubbedCmds}, prompter)

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

	t.Run("it should determine the public path at runtime if not given", func(t *testing.T) {
		var tests = []struct {
			name           string
			stubbedCmds    map[string]string
			promptCalls    int
			wantPublicPath types.PublicPath
		}{
			{
				"find 1 public path",
				map[string]string{"find -L . -type f -name 'wp-config.php'": "./public/wp-config.php\n"},
				0,
				"./public",
			},
			{
				"via prompter if cmd cannot be run",
				map[string]string{},
				1,
				"./path/to/public",
			},
			{
				"no public path found - prompt",
				map[string]string{"find -L . -type f -name 'wp-config.php'": ""},
				1,
				"./path/to/public",
			},
			{
				"multiple public paths found - use none and prompt",
				map[string]string{"find -L . -type f -name 'wp-config.php'": "./public/wp-config.php\n./public2/wp-config.php\n"},
				1,
				"./path/to/public",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				prompter := &PrompterSpy{}

				got, err := DetermineSiteInfo("localhost", "", newConfigParserStub(), &MockCommandRunner{tt.stubbedCmds}, prompter)

				if err != nil {
					t.Errorf("got error %v; want nil", err)
				}

				want := SiteInfo{"localhost", tt.wantPublicPath, database.DatabaseCredentials{"user", "pass", "db", "localhost"}}
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

type ConfigParserStub struct {
	errorStub  error
	fieldsStub parser.WPConfigFields
}

func newConfigParserStub() *ConfigParserStub {
	return &ConfigParserStub{
		fieldsStub: parser.WPConfigFields{Credentials: database.DatabaseCredentials{"user", "pass", "db", "localhost"}, Prefix: "wp_"},
	}
}

func (p *ConfigParserStub) ParseWPConfig(publicPath types.PublicPath) (parser.WPConfigFields, error) {
	return p.fieldsStub, p.errorStub
}

type PrompterSpy struct {
	calls int
}

func (p *PrompterSpy) Prompt(question string) string {
	p.calls++

	switch question {
	case "What is the site url?":
		return "http://prompted-localhost"
	case "What is the public path?":
		return "./path/to/public"
	}

	return ""
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
