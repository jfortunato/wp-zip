VERSION = $(shell grep -Po 'v[0-9]+\.[0-9]+\.[0-9]+' version.go)
GITCOMMIT=$(shell git rev-parse --short HEAD)
BUILDTIME=$(shell date -u --iso-8601=seconds)

# Strip debug info
GO_LDFLAGS = -s -w

# Add some build information
GO_LDFLAGS += -X 'main.commit=$(GITCOMMIT)'
GO_LDFLAGS += -X 'main.date=$(BUILDTIME)'

.PHONY: binary
binary: build/wp-zip

build/wp-zip:
	go build -ldflags "$(GO_LDFLAGS)" -o $@

.PHONY: platform-all
platform-all: platform-linux platform-darwin platform-windows

.PHONY: platform-linux
platform-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o build/wp-zip-linux-amd64/wp-zip
	GOOS=linux GOARCH=386 go build -ldflags "$(GO_LDFLAGS)" -o build/wp-zip-linux-386/wp-zip
	GOOS=linux GOARCH=arm64 go build -ldflags "$(GO_LDFLAGS)" -o build/wp-zip-linux-arm64/wp-zip

.PHONY: platform-darwin
platform-darwin:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o build/wp-zip-darwin-amd64/wp-zip
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(GO_LDFLAGS)" -o build/wp-zip-darwin-arm64/wp-zip

.PHONY: platform-windows
platform-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o build/wp-zip-windows-amd64/wp-zip.exe
	GOOS=windows GOARCH=386 go build -ldflags "$(GO_LDFLAGS)" -o build/wp-zip-windows-386/wp-zip.exe
	GOOS=windows GOARCH=arm64 go build -ldflags "$(GO_LDFLAGS)" -o build/wp-zip-windows-arm64/wp-zip.exe

.PHONY: release
release: platform-linux platform-darwin platform-windows
	mkdir build/release
	cd build && tar -zcf release/wp-zip-$(VERSION)-Linux_x86_64.tar.gz wp-zip-linux-amd64
	cd build && tar -zcf release/wp-zip-$(VERSION)-Linux_i386.tar.gz wp-zip-linux-386
	cd build && tar -zcf release/wp-zip-$(VERSION)-Linux_arm64.tar.gz wp-zip-linux-arm64
	cd build && tar -zcf release/wp-zip-$(VERSION)-Darwin_x86_64.tar.gz wp-zip-darwin-amd64
	cd build && tar -zcf release/wp-zip-$(VERSION)-Darwin_arm64.tar.gz wp-zip-darwin-arm64
	cd build && zip -r release/wp-zip-$(VERSION)-Windows_x86_64.zip wp-zip-windows-amd64
	cd build && zip -r release/wp-zip-$(VERSION)-Windows_i386.zip wp-zip-windows-386
	cd build && zip -r release/wp-zip-$(VERSION)-Windows_arm64.zip wp-zip-windows-arm64
	cd build/release && sha256sum wp-zip-$(VERSION)-Linux_x86_64.tar.gz >> checksums.txt
	cd build/release && sha256sum wp-zip-$(VERSION)-Linux_i386.tar.gz >> checksums.txt
	cd build/release && sha256sum wp-zip-$(VERSION)-Linux_arm64.tar.gz >> checksums.txt
	cd build/release && sha256sum wp-zip-$(VERSION)-Darwin_x86_64.tar.gz >> checksums.txt
	cd build/release && sha256sum wp-zip-$(VERSION)-Darwin_arm64.tar.gz >> checksums.txt
	cd build/release && sha256sum wp-zip-$(VERSION)-Windows_x86_64.zip >> checksums.txt
	cd build/release && sha256sum wp-zip-$(VERSION)-Windows_i386.zip >> checksums.txt
	cd build/release && sha256sum wp-zip-$(VERSION)-Windows_arm64.zip >> checksums.txt

.PHONY: clean
clean:
	rm -rf build
