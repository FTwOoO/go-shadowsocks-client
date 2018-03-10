RELEASE_PATH = release
PACKAGE_PATH = release/gsc

build:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $(RELEASE_PATH)/gsc

package:
	rm -Rf $(PACKAGE_PATH)/*
	mkdir -p $(PACKAGE_PATH)
	cp ./geoip.mmdb $(PACKAGE_PATH)
	# macOS
	cp ./release/gsc-darwin-amd64 $(PACKAGE_PATH)
	cd ./release && zip gsc-darwin-amd64.zip gsc

	# remove history
	rm $(PACKAGE_PATH)gsc

