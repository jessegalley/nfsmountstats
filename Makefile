export CGO_ENABLED=0

build:
	        go build -o bin/nfsperfmon cmd/nfsperfmon/main.go

run: build
	        ./bin/nfsperfmon

test:
	        go test -v ./... 
