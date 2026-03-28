BINARY=winmole
LDFLAGS=-ldflags="-s -w"

.PHONY: build build-local build-windows-amd64 build-windows-arm64 fmt vet test clean

build: build-windows-amd64

build-local:
	go build $(LDFLAGS) -o bin\$(BINARY).exe .

build-windows-amd64:
	set GOOS=windows& set GOARCH=amd64& go build $(LDFLAGS) -o bin\$(BINARY)-windows-amd64.exe .

build-windows-arm64:
	set GOOS=windows& set GOARCH=arm64& go build $(LDFLAGS) -o bin\$(BINARY)-windows-arm64.exe .

fmt:
	gofmt -w main.go cmd internal pkg

vet:
	go vet ./...

test:
	go test ./...

clean:
	-del /Q bin\$(BINARY)*.exe 2>nul
