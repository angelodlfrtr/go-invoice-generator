.PHONY: lint
lint:
	go vet ./...
	golangci-lint run

.PHONY: fmt
fmt:
	dprint fmt

.PHONY: fmt-check
fmt-check:
	dprint check

.PHONY: gosec
gosec:
	gosec ./...

.PHONY: test
test:
	go test -count 1 ./...

.PHONY: testwithcover
testwithcover:
	go test -count 1 --coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: verapdf-check-3b
verapdf-check-3b: test
	verapdf --flavour 3b out/facturx.pdf
	verapdf --flavour 3b out/facturx_MINIMUM.pdf
	verapdf --flavour 3b out/facturx_EXTENDED.pdf
	verapdf --flavour 3b out/facturx_EN_16931.pdf
	verapdf --flavour 3b out/facturx_BASIC-WL.pdf
	verapdf --flavour 3b out/facturx_BASIC.pdf

.PHONY: mustang-check-facturx
mustang-check-facturx: test
	mustang-cli --action validate --source out/facturx.pdf
	mustang-cli --action validate --source out/facturx_MINIMUM.pdf
	mustang-cli --action validate --source out/facturx_EXTENDED.pdf
	mustang-cli --action validate --source out/facturx_EN_16931.pdf
	mustang-cli --action validate --source out/facturx_BASIC-WL.pdf
	mustang-cli --action validate --source out/facturx_BASIC.pdf

.PHONY: clean
clean:
	rm -Rf coverage.out out/
