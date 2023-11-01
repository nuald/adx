setupCore:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
		| sh -s -- -b $(shell go env GOPATH)/bin v1.55.1
	go get -u \
		github.com/go-errors/errors \
		github.com/go-bindata/go-bindata/... \
		gopkg.in/yaml.v2

assets: setupCore
	go-bindata data/

lint: assets
	golangci-lint run

unitTest:
	go test -v -cover ./...

gofmt:
	gofmt -s -w .

tests: gofmt lint unitTest
	@echo "tests"

install: tests
	go install
