RELEASE_PATH = release
PACKAGE_PATH = release/go-ss-client

install:
	@go get
build:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $(RELEASE_PATH)/go-ss-client-darwin-amd64
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o $(RELEASE_PATH)/go-ss-client-amd64
	GOOS=linux GOARCH=386 go build -ldflags "-s -w" -o $(RELEASE_PATH)/go-ss-client-386
	GOOS=windows GOARCH=386 go build -ldflags "-s -w" -o $(RELEASE_PATH)/go-ss-client-386.exe
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o $(RELEASE_PATH)/go-ss-client-amd64.exe
package:
	rm -Rf $(PACKAGE_PATH)/*
	mkdir -p $(PACKAGE_PATH)
	cp ./go-ss-client.default.conf $(PACKAGE_PATH)
	cp ./geoip.mmdb $(PACKAGE_PATH)
	cp ./LICENSE $(RELEASE_PATH)
	cp ./README.md $(PACKAGE_PATH)
	# macOS
	cp ./release/go-ss-client-darwin-amd64 $(PACKAGE_PATH)
	cd ./release && zip go-ss-client-darwin-amd64.zip go-ss-client
	# Linux amd64
	cp ./release/go-ss-client-amd64 $(PACKAGE_PATH)go-ss-client
	cd ./release && tar zcf go-ss-client-linux-amd64.tar.gz go-ss-client
	# Linux 386
	cp ./release/go-ss-client-386 $(PACKAGE_PATH)
	cd ./release && tar zcf go-ss-client-linux-386.tar.gz go-ss-client
	# Windows 386
    #cp ./release/go-ss-client-386.exe $(PACKAGE_PATH)
    #cd ./release && tar zcf go-ss-client-win-386.tar.gz go-ss-client
    # Windows amd64
    #cp ./release/go-ss-client-amd64.exe $(PACKAGE_PATH)
    #cd ./release && tar zcf go-ss-client-win-amd64.tar.gz go-ss-client
	# remove history
	rm $(PACKAGE_PATH)go-ss-client
run:
	@go run main.go
test:
	@go test ./go-ss-client
