package main

import cmd "github.com/jfortunato/wp-zip/cmd/wp-zip"

var (
	commit = "none"
	date   = "unknown"
)

func main() {
	cmd.Execute(cmd.VersionDetails{
		Version: version,
		Commit:  commit,
		Date:    date,
	})
}
