lint:
	golangci-lint run

test:
	go test -count 1 ./...

testwithcover:
	go test -count 1 --coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -Rf coverage.out
