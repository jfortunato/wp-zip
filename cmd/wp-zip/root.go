package wp_zip

import (
	"errors"
	"github.com/jfortunato/wp-zip/internal/builder"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"github.com/spf13/cobra"
	"log"
)

type VersionDetails struct {
	Version string
	Commit  string
	Date    string
}

var Host string
var Username string
var Password string
var Port string
var Domain string
var Webroot string

func init() {
	// Override the help command to disable the shorthand -h, as it is used for the --host instead
	rootCmd.Flags().BoolP("help", "", false, "help for this command")
	rootCmd.Flags().StringVarP(&Host, "host", "h", "", "SFTP host (required)")
	rootCmd.Flags().StringVarP(&Username, "username", "u", "", "SFTP username (required)")
	rootCmd.Flags().StringVarP(&Password, "password", "p", "", "SFTP password (required)")
	rootCmd.Flags().StringVarP(&Port, "port", "P", "22", "SFTP port")
	rootCmd.Flags().StringVarP(&Domain, "domain", "d", "", "Domain name of the live site")
	rootCmd.Flags().StringVarP(&Webroot, "webroot", "w", "", "Path to the public directory of the live site")
	rootCmd.MarkFlagRequired("host")
	rootCmd.MarkFlagRequired("username")
	// TODO: Make this optional, and prompt for it if not provided
	rootCmd.MarkFlagRequired("password")
	rootCmd.MarkFlagRequired("domain")
	rootCmd.MarkFlagRequired("webroot")

}

var rootCmd = &cobra.Command{
	Use:   "wp-zip -h sftp-host -u sftp-username -p sftp-password -d example.com -w path-to-webroot [flags] output-filename",
	Short: "Export an existing WordPress site to a zip file",
	Long: `Generate a complete archive of a WordPress site's files
	and database, which can be used to migrate the site
	to another host or to create a local development environment.`,
	Args: argsValidation(),
	Run: func(cmd *cobra.Command, args []string) {
		err := builder.PackageWP(sftp.SSHCredentials{User: Username, Pass: Password, Host: Host, Port: Port}, builder.Domain(Domain), builder.PublicPath(Webroot), args[0])
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func argsValidation() cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		// Must have exactly one argument
		if len(args) != 1 {
			return errors.New("requires exactly one argument")
		}

		return nil
	}
}

func Execute(v VersionDetails) {
	rootCmd.Version = v.Version

	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
