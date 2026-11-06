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

# mustang-cli exits 0 even when the <pdf> or <xml> sub-summary is invalid,
# so inspect the report for invalid statuses and error elements.
.PHONY: mustang-check-facturx
mustang-check-facturx: test
	@for f in out/facturx.pdf out/facturx_MINIMUM.pdf out/facturx_EXTENDED.pdf \
		out/facturx_EN_16931.pdf out/facturx_BASIC-WL.pdf out/facturx_BASIC.pdf; do \
		echo "== $$f"; \
		report=$$(mustang-cli --action validate --source "$$f"); echo "$$report"; \
		if echo "$$report" | grep -q -e 'status="invalid"' -e '<error'; then \
			echo "mustang: $$f is NOT valid"; exit 1; \
		fi; \
	done

.PHONY: clean
clean:
	rm -Rf coverage.out out/
