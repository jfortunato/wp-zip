package parser

import (
	"errors"
	"github.com/jfortunato/wp-zip/internal/database"
	"github.com/jfortunato/wp-zip/internal/emitter"
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	t.Run("it should parse the database credentials from the wp-config.php file", func(t *testing.T) {
		contents := `
<?php
define('DB_NAME', 'name');
define('DB_USER', 'user');
define('DB_PASSWORD', 'pass');
define('DB_HOST', 'localhost');

$table_prefix = 'wp_';
`

		parser := NewEmitterCredentialsParser(&EmitterStub{contentsToEmit: contents})

		fields, err := parser.ParseWPConfig("/var/www/html/")

		if err != nil {
			t.Errorf("got error %v; want nil", err)
		}

		expectedCreds := database.DatabaseCredentials{User: "user", Pass: "pass", Name: "name", Host: "localhost"}

		if fields.Credentials != expectedCreds {
			t.Errorf("got %v; want %v", fields.Credentials, expectedCreds)
		}
	})

	t.Run("it should parse the prefix from the wp-config.php file", func(t *testing.T) {
		contents := `
<?php
define('DB_NAME', 'name');
define('DB_USER', 'user');
define('DB_PASSWORD', 'pass');
define('DB_HOST', 'localhost');

$table_prefix  = 'foo_';
`

		parser := NewEmitterCredentialsParser(&EmitterStub{contentsToEmit: contents})

		fields, err := parser.ParseWPConfig("/var/www/html/")

		if err != nil {
			t.Errorf("got error %v; want nil", err)
		}

		expectedPrefix := "foo_"

		if fields.Prefix != expectedPrefix {
			t.Errorf("got %v; want %v", fields.Prefix, expectedPrefix)
		}
	})

	t.Run("it should return an error if it cannot read the wp-config.php file", func(t *testing.T) {
		var tests = []struct {
			name            string
			emitterError    error
			emitterContents string
		}{
			{"emitter error", errors.New("emitter error"), "doesnt matter"},
			{"empty contents", nil, ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				parser := NewEmitterCredentialsParser(&EmitterStub{errorStub: tt.emitterError})

				_, err := parser.ParseWPConfig("/var/www/html/")

				if !errors.Is(err, ErrCouldNotReadWPConfig) {
					t.Errorf("got error %v; want ErrCouldNotReadWPConfig", err)
				}
			})
		}
	})

	t.Run("it should return an error if the credentials cant be extracted from the file", func(t *testing.T) {
		parser := NewEmitterCredentialsParser(&EmitterStub{contentsToEmit: "some contents that don't contain creds"})

		_, err := parser.ParseWPConfig("/var/www/html/")

		if !errors.Is(err, ErrCantFindCredentials) {
			t.Errorf("got error %v; want ErrCantFindCredentials", err)
		}
	})

	t.Run("it should return an error if the credentials can be extracted from the file, but the prefix cannot", func(t *testing.T) {
		contents := `
<?php
define('DB_NAME', 'name');
define('DB_USER', 'user');
define('DB_PASSWORD', 'pass');
define('DB_HOST', 'localhost');
`
		parser := NewEmitterCredentialsParser(&EmitterStub{contentsToEmit: contents})

		_, err := parser.ParseWPConfig("/var/www/html/")

		if !errors.Is(err, ErrCantFindPrefix) {
			t.Errorf("got error %v; want ErrCantFindPrefix", err)
		}
	})

	t.Run("it can handle many variations", func(t *testing.T) {
		var tests = []struct {
			name     string
			contents string
			expected database.DatabaseCredentials
		}{
			{
				"basic",
				`<?php
				define('DB_USER', 'user');
				define('DB_PASSWORD', 'pass');
				define('DB_NAME', 'dbname');
				define('DB_HOST', 'localhost');
				`,
				database.DatabaseCredentials{"user", "pass", "dbname", "localhost"},
			},
			{
				"spaces",
				`<?php
				// Before/after opening/closing parenthesis
				define( 'DB_USER', 'user' );
				// Extra spaces
				define(   'DB_PASSWORD',   'pass'   );
				// No spaces
				define('DB_NAME','dbname');
				define('DB_HOST','localhost');
				`,
				database.DatabaseCredentials{"user", "pass", "dbname", "localhost"},
			},
			{
				"double quotes",
				`<?php
				define( "DB_USER", "user" );
				define( "DB_PASSWORD", "pass" );
				define( "DB_NAME", "dbname" );
				define( "DB_HOST", "localhost" );
				`,
				database.DatabaseCredentials{"user", "pass", "dbname", "localhost"},
			},
			{
				"quote usage in values",
				`<?php
				define('DB_USER', 'us"er');
				define('DB_PASSWORD', "pa'ss");
				define('DB_NAME', 'dbname');
				define('DB_HOST', 'localhost');
				`,
				database.DatabaseCredentials{"us\"er", "pa'ss", "dbname", "localhost"},
			},
			//{
			//	"docker env",
			//	`<?php
			//	define('DB_USER', getenv_docker('WORDPRESS_DB_USER', 'default-user'));
			//	define('DB_PASSWORD', getenv_docker('WORDPRESS_DB_PASSWORD', 'default-pass'));
			//	define('DB_NAME', getenv_docker('WORDPRESS_DB_NAME', 'default-dbname'));
			//	define('DB_HOST', getenv_docker('WORDPRESS_DB_HOST', 'default-localhost'));
			//	`,
			//	DatabaseCredentials{"user", "pass", "dbname", "localhost"},
			//},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				// Add the prefix to the contents, we are only testing the credentials here
				contents := test.contents + "\n$table_prefix  = 'wp_';"

				parser := NewEmitterCredentialsParser(&EmitterStub{contentsToEmit: contents})

				fields, err := parser.ParseWPConfig("/var/www/html/")

				if err != nil {
					t.Errorf("got error %v; want nil", err)
				}

				// Assert that the credentials are what we expect
				if fields.Credentials != test.expected {
					t.Errorf("got %v; want %v", fields.Credentials, test.expected)
				}
			})
		}
	})
}

type EmitterStub struct {
	contentsToEmit string
	errorStub      error
}

func (e *EmitterStub) EmitSingle(src string, fn emitter.EmitFunc) error {
	fn(src, strings.NewReader(e.contentsToEmit))

	return e.errorStub
}
