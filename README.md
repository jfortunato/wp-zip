# wp-zip

![Release Version](https://img.shields.io/github/v/release/jfortunato/wp-zip)
[![Tests Status](https://github.com/jfortunato/wp-zip/actions/workflows/run-tests.yaml/badge.svg
)](https://github.com/jfortunato/wp-zip/actions/workflows/run-tests.yaml)
[![codecov](https://codecov.io/gh/jfortunato/wp-zip/graph/badge.svg?token=ZNZN0I2Y52)](https://codecov.io/gh/jfortunato/wp-zip)
[![Go Report Card](https://goreportcard.com/badge/github.com/jfortunato/wp-zip)](https://goreportcard.com/report/github.com/jfortunato/wp-zip)

Quickly archive a live WordPress site to a local zip file.

## Installation

### via Prebuilt Binaries

Prebuilt binaries are available for Linux, macOS, and Windows. You can download them from the [releases page](https://github.com/jfortunato/wp-zip/releases/latest).

### via Go Install

If you have Go installed, you can install `wp-zip` with the following command:

```bash
go install github.com/jfortunato/wp-zip@latest
```

## Usage

```bash
wp-zip -h <sftp-host> -u <sftp-user> output.zip
```

You will be prompted for the sftp password (if `-p` flag not given). You must already have access to the site via SFTP. The path to the public directory (where wp-config.php lives) should be automatically detected, but if it can't, you will be prompted for it.

## Importing with LocalWP

Once you have a zip file, you can import it into [LocalWP](https://localwp.com/). This makes it very easy to quickly get up and running with a local WordPress site.
