# wp-zip

Quickly archive a live WordPress site to a local zip file.

## Usage

```bash
wp-zip -h <sftp-host> -u <sftp-user> -p <sftp-password> -d <domain> path/to/public/
```

You must already have access to the site via SFTP. The -d flag represents the live domain name of the site, for example 'example.com' The path to the public directory is the path to the WordPress installation (the directory that wp-config.php lives under).

## Importing with LocalWP

Once you have a zip file, you can import it into [LocalWP](https://localwp.com/). This makes it very easy to quickly get up and running with a local WordPress site.
