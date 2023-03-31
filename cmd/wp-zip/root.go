package wp_zip

import (
	"errors"
	"fmt"
	"github.com/jfortunato/wp-zip/internal/packager"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"github.com/spf13/cobra"
	"os"
)

var Host string
var Username string
var Password string
var Port string

func init() {
	// Override the help command to disable the shorthand -h, as it is used for the --host instead
	rootCmd.Flags().BoolP("help", "", false, "help for this command")
	rootCmd.Flags().StringVarP(&Host, "host", "h", "", "SFTP host (required)")
	rootCmd.Flags().StringVarP(&Username, "username", "u", "", "SFTP username (required)")
	rootCmd.Flags().StringVarP(&Password, "password", "p", "", "SFTP password (required)")
	rootCmd.Flags().StringVarP(&Port, "port", "P", "22", "SFTP port")
	rootCmd.MarkFlagRequired("host")
	rootCmd.MarkFlagRequired("username")
	rootCmd.MarkFlagRequired("password")

}

var rootCmd = &cobra.Command{
	Use:     "wp-zip -h sftp-host -u sftp-username -p sftp-password [flags] PATH_TO_PUBLIC",
	Version: "0.0.1",
	Short:   "Export an existing WordPress site to a zip file",
	Long: `Generate a complete archive of a WordPress site's files
	and database, which can be used to migrate the site
	to another host or to create a local development environment.`,
	Args: argsValidation(),
	Run: func(cmd *cobra.Command, args []string) {
		packager.PackageWP(sftp.SSHCredentials{User: Username, Pass: Password, Host: Host, Port: Port}, args[0])
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
