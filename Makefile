.PHONY: fmt test upgrade

fmt:
	gofumpt -l -w .

test:
	go test ./...

upgrade:
	go get -u ./...
	go mod tidy
