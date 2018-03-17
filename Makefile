RELEASE_PATH = release
PACKAGE_PATH = release/gss

build:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $(RELEASE_PATH)/gsc


