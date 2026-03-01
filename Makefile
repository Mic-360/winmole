.PHONY: build build-local clean

LDFLAGS = -ldflags="-s -w"

build:
	set GOOS=windows& set GOARCH=amd64& go build $(LDFLAGS) -o bin/analyze-go.exe ./cmd/analyze
	set GOOS=windows& set GOARCH=amd64& go build $(LDFLAGS) -o bin/status-go.exe  ./cmd/status

build-local:
	go build $(LDFLAGS) -o bin/analyze-go.exe ./cmd/analyze
	go build $(LDFLAGS) -o bin/status-go.exe  ./cmd/status

clean:
	-del /Q bin\*.exe 2>nul
