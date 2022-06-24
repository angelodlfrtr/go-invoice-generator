.PHONY: lint
lint:
	go vet ./...
	golangci-lint run

.PHONY: test
test:
	go test -count 1 ./...

.PHONY: testwithcover
testwithcover:
	go test -count 1 --coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: clean
clean:
	rm -Rf coverage.out
