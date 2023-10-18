# wp-zip

Quickly archive a live WordPress site to a local zip file.

## Usage

```bash
wp-zip -h <sftp-host> -u <sftp-user> -p <sftp-password> output.zip
```

You must already have access to the site via SFTP. The path to the public directory (where wp-config.php lives) should be automatically detected, but if it can't, you will be prompted for it.

## Importing with LocalWP

Once you have a zip file, you can import it into [LocalWP](https://localwp.com/). This makes it very easy to quickly get up and running with a local WordPress site.
