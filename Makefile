.PHONY: all clean

all: build-linux-amd64 build-linux-386 build-windows-amd64 build-windows-386 build-darwin-amd64 build-darwin-arm64 build-linux-arm64 build-linux-arm build-windows-arm build-windows-arm64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/tracker-linux-x64

build-linux-386:
	GOOS=linux GOARCH=386 go build -ldflags="-s -w" -o bin/tracker-linux-x32

build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/tracker-windows-x64.exe

build-windows-386:
	GOOS=windows GOARCH=386 go build -ldflags="-s -w" -o bin/tracker-windows-x32.exe

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/tracker-mac-x64

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/tracker-mac-arm64

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/tracker-linux-arm64

build-linux-arm:
	GOOS=linux GOARCH=arm go build -ldflags="-s -w" -o bin/tracker-linux-arm32

build-windows-arm64:
	GOOS=windows GOARCH=arm64 go build -ldflags="-s -w" -o bin/tracker-windows-arm

build-windows-arm:
	GOOS=windows GOARCH=arm go build -ldflags="-s -w" -o bin/tracker-windows-arm64

clean:
	rm -rf bin

