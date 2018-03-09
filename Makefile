RELEASE_PATH = release
PACKAGE_PATH = release/gss

build:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $(RELEASE_PATH)/gss

package:
	rm -Rf $(PACKAGE_PATH)/*
	mkdir -p $(PACKAGE_PATH)
	cp ./geoip.mmdb $(PACKAGE_PATH)
	# macOS
	cp ./release/gss-darwin-amd64 $(PACKAGE_PATH)
	cd ./release && zip gss-darwin-amd64.zip gss

	# remove history
	rm $(PACKAGE_PATH)gss

