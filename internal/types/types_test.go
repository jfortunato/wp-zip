package types

import "testing"

func TestSiteUrl(t *testing.T) {
	t.Run("it can be created from the constructor", func(t *testing.T) {
		var tests = []struct {
			name     string
			input    string
			expected SiteUrl
		}{
			{"full https url", "https://example.com", "https://example.com"},
			{"full http url", "http://example.com", "http://example.com"},
			{"full url trailing slash", "https://example.com/", "https://example.com"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				url, err := NewSiteUrl(test.input)

				if err != nil {
					t.Errorf("got error %v; want nil", err)
				}

				if url != test.expected {
					t.Errorf("got %v; want %v", url, test.expected)
				}
			})
		}
	})

	t.Run("its constructor should return an error for an invalid url", func(t *testing.T) {
		var tests = []struct {
			name  string
			input string
		}{
			{"missing protocol", "example.com"},
			{"nonsense", "xlkjad-lk"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				_, err := NewSiteUrl(test.input)

				if err == nil {
					t.Errorf("got nil; want error")
				}
			})
		}
	})

	t.Run("its Domain method should remove the protocol", func(t *testing.T) {
		var tests = []struct {
			name     string
			input    string
			expected string
		}{
			{"https", "https://example.com", "example.com"},
			{"http", "http://example.com", "example.com"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				url, err := NewSiteUrl(test.input)

				if err != nil {
					t.Errorf("got error %v; want nil", err)
				}

				if url.Domain() != test.expected {
					t.Errorf("got %s; want %s", url.Domain(), test.expected)
				}
			})
		}
	})
}

func TestPublicPath_String(t *testing.T) {
	t.Run("it should always end in forward slash when converting to string", func(t *testing.T) {
		var tests = []struct {
			input    string
			expected string
		}{
			{"/srv/", "/srv/"},
			{"/srv", "/srv/"},
			{"/", "/"},
		}

		for _, test := range tests {
			path := PublicPath(test.input)
			if path.String() != test.expected {
				t.Errorf("got %s; want %s", path.String(), test.expected)
			}
		}
	})
}
