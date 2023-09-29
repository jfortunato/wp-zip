package operations

import (
	"encoding/json"
	"fmt"
	"github.com/jfortunato/wp-zip/internal/database"
	"github.com/jfortunato/wp-zip/internal/sftp"
	"github.com/jfortunato/wp-zip/internal/types"
	"github.com/pkg/errors"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

var (
	ErrCouldNotUploadFile = errors.New("could not upload file")
	ErrCouldNotDeleteFile = errors.New("error deleting file")
	ErrInvalidResponse    = errors.New("invalid response from server")
	ErrUnexpectedResponse = errors.New("unexpected response from server")
)

type HttpGetter interface {
	Get(url string) (resp io.ReadCloser, err error)
}

type GenerateJsonOperation struct {
	u               sftp.FileUploadDeleter
	g               HttpGetter
	publicUrl       types.Domain
	publicPath      types.PublicPath
	credentials     database.DatabaseCredentials
	randomFileNamer func() string
}

func NewGenerateJsonOperation(u sftp.FileUploadDeleter, g HttpGetter, publicUrl types.Domain, publicPath types.PublicPath, credentials database.DatabaseCredentials) *GenerateJsonOperation {
	return &GenerateJsonOperation{u, g, publicUrl, publicPath, credentials, func() string { return "wp-zip-" + randSeq(10) + ".php" }}
}

func (o *GenerateJsonOperation) SendFiles(fn SendFilesFunc) (err error) {
	// We need to:
	// 1. Upload our custom PHP file to the server
	// 2. Make an HTTP request to the file, which generates the JSON content we need
	// 3. Send the JSON content back to the caller with the SendFilesFunc
	// 4. Delete the file we uploaded from the server

	// 1.
	// Generate a random filename
	basename := o.randomFileNamer()
	uploadFilename := string(o.publicPath) + "/" + basename
	err = o.u.Upload(strings.NewReader(getPhpFileContents(o.credentials, o.publicUrl, o.publicPath)), uploadFilename)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCouldNotUploadFile, err)
	}
	// 4.
	// Use defer to ensure that the file is deleted even if there is an error
	defer func() {
		errD := o.u.Delete(uploadFilename)
		if errD != nil {
			err = fmt.Errorf("%w: %s", ErrCouldNotDeleteFile, errD)
		}
	}()

	// 2.
	resp, err := o.g.Get(o.publicUrl.AsSecureUrl() + "/" + basename)
	if err != nil {
		// Try an insecure URL before returning an error
		resp, err = o.g.Get(o.publicUrl.AsInsecureUrl() + "/" + basename)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidResponse, err)
		}
	}
	defer resp.Close()
	// Read the response body to a string
	buf := new(strings.Builder)
	_, err = io.Copy(buf, resp)
	contents := buf.String()
	// Assert that the json response contains the key "name"
	if !assertResponseContainsJsonKey(contents, "name") {
		return ErrUnexpectedResponse
	}

	// 3.
	return fn(File{
		Name: "wpmigrate-export.json",
		Body: strings.NewReader(contents),
	})
}

func assertResponseContainsJsonKey(response, key string) bool {
	var jsonResp map[string]interface{}
	err := json.Unmarshal([]byte(response), &jsonResp)
	if err != nil {
		//return errors.Wrap(err, "could not unmarshal json")
		return false
	}
	if _, ok := jsonResp["name"]; !ok {
		return false
	}

	return true
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	rand.Seed(time.Now().UnixNano())
	//r := rand.New(rand.NewSource(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func getPhpFileContents(credentials database.DatabaseCredentials, publicUrl types.Domain, publicPath types.PublicPath) string {
	return fmt.Sprintf(`<?php

// Connect to mysql and get the mysql version
$link = mysqli_connect('localhost', '%s', '%s', '%s');
$mysqlVersion = mysqli_get_server_info($link);
mysqli_close($link);

// Get the server name and version
preg_match('/^(apache|nginx)\/(\d+\.\d+\.\d+).*/', strtolower($_SERVER['SERVER_SOFTWARE']), $matches);
$serverJson = isset($matches[1], $matches[2]) ? [ $matches[1] => [ 'name' => $matches[1], 'version' => $matches[2] ] ] : '';

// Get the current WordPress version by reading the wp-includes/version.php file
$wpVersionFile = file_get_contents(__DIR__ . DIRECTORY_SEPARATOR . 'wp-includes' . DIRECTORY_SEPARATOR .  'version.php');
preg_match('/\$wp_version = \'(.*)\';/', $wpVersionFile, $matches);
$wpVersion = isset($matches[1]) ? $matches[1] : '';

header('Content-Type: application/json');
echo json_encode(array_merge_recursive([
    'name' => '%s',
    'domain' => '%s',
    'path' => '%s',
    'wpVersion' => $wpVersion,
    'services' => [
        'php' => [
            'name' => 'php',
            'version' => PHP_VERSION,
        ],
        'mysql' => [
            'name' => 'mysql',
            'version' => $mysqlVersion,
        ],
    ],
], ['services' => $serverJson]));
`, credentials.User, credentials.Pass, credentials.Name, publicUrl, publicUrl, publicPath)
}

type BasicHttpGetter struct{}

func (g *BasicHttpGetter) Get(url string) (resp io.ReadCloser, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "error making http request")
	}

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error reading response body")
	}

	return io.NopCloser(strings.NewReader(string(body))), nil
}
