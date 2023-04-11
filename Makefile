WP_ZIP_VERSION = $(shell cat version.go | grep -oP '(?<=const wpZipVersion = ")[^"]*' | head -n 1)

# Strip debug info
GO_FLAGS += "-ldflags=-s -w"

.PHONY: wp-zip clean platform-linux platform-windows platform-darwin

wp-zip:
	go build -o build/wp-zip

platform-all: platform-linux platform-windows platform-darwin

platform-linux:
	GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o build/wp-zip-v$(WP_ZIP_VERSION)-linux-amd64.exe

platform-windows:
	GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) -o build/wp-zip-v$(WP_ZIP_VERSION)-windows-amd64.exe

platform-darwin:
	GOOS=darwin GOARCH=amd64 go build $(GO_FLAGS) -o build/wp-zip-v$(WP_ZIP_VERSION)-darwin-amd64

clean:
	rm -rf build
