package wp_zip

import (
	"errors"
	"github.com/jfortunato/wp-zip/internal/packager"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"github.com/jfortunato/wp-zip/internal/types"
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
var SiteUrl string
var Webroot string

// RunOptions are the pre-run validated options that are passed to the Run function
type RunOptions struct {
	sshCredentials sftp.SSHCredentials
	siteUrl        types.SiteUrl
	publicPath     types.PublicPath
}

var Options RunOptions

func init() {
	// Override the help command to disable the shorthand -h, as it is used for the --host instead
	rootCmd.Flags().BoolP("help", "", false, "help for this command")
	rootCmd.Flags().StringVarP(&Host, "host", "h", "", "SFTP host (required)")
	rootCmd.Flags().StringVarP(&Username, "username", "u", "", "SFTP username (required)")
	rootCmd.Flags().StringVarP(&Password, "password", "p", "", "SFTP password (required or prompted)")
	rootCmd.Flags().StringVarP(&Port, "port", "P", "22", "SFTP port")
	rootCmd.Flags().StringVarP(&SiteUrl, "site-url", "", "", "Site url name of the live site, including the protocol (e.g. https://example.com)")
	rootCmd.Flags().StringVarP(&Webroot, "webroot", "w", "", "Path to the public directory of the live site")
	rootCmd.MarkFlagRequired("host")
	rootCmd.MarkFlagRequired("username")
	rootCmd.MarkFlagRequired("webroot")

}

var rootCmd = &cobra.Command{
	Use:   "wp-zip -h sftp-host -u sftp-username -p sftp-password -w path-to-webroot [flags] output-filename",
	Short: "Export an existing WordPress site to a zip file",
	Long: `Generate a complete archive of a WordPress site's files
	and database, which can be used to migrate the site
	to another host or to create a local development environment.`,
	Args: argsValidation(),
	PreRun: func(cmd *cobra.Command, args []string) {
		var err error

		// If the user didn't provide a password, then prompt for it
		// Run in a loop until the user provides a password
		for Password == "" {
			prompter := &packager.RuntimePrompter{}
			Password = prompter.PromptForPassword("Enter SFTP password: ")
		}

		// If the user supplied a site url, make sure it is valid
		var siteUrl types.SiteUrl
		if SiteUrl != "" {
			siteUrl, err = types.NewSiteUrl(SiteUrl)
			if err != nil {
				log.Fatalln(err)
			}
		}

		// Construct all the RunOptions
		Options = RunOptions{
			sftp.SSHCredentials{User: Username, Pass: Password, Host: Host, Port: Port},
			siteUrl,
			types.PublicPath(Webroot),
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		p, err := packager.NewPackager(Options.sshCredentials, Options.siteUrl, Options.publicPath)
		if err != nil {
			log.Fatalln(err)
		}

		err = p.PackageWP(args[0])
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
