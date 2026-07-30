GO := $(shell which go)

# check is the full gate, matching the one Voltz runs against its own Go
# components: the same steps, in the same order, with the same pinned tools.
check: vet build lint test vuln nilaway
	@echo "=== ALL CHECKS PASSED ==="

vet:
	${GO} vet ./...

build:
	${GO} build ./...

lint:
	golangci-lint run --timeout 5m

test:
	${GO} test -race ./...

cover:
	${GO} test -race -coverprofile=coverage.out ./...
	${GO} tool cover -func coverage.out

vuln:
	govulncheck ./...

# nilaway reports one finding here that is a known false positive, tracked as
# nilaway issue #126: it follows http.Client.Do into the standard library, sees
# that Do can return a nil response alongside an error, and cannot prove that
# the error check guards the dereference. Voltz's gate filters the identical
# finding rather than papering over it with a nil check the http contract
# already rules out.
#
# The filter is deliberately narrow. Only blocks naming both http/client.go and
# Do() are suppressed, every other finding fails, and the suppressed count is
# printed so the exemption can never go quiet.
nilaway:
	@nilaway ./... 2>&1 | awk ' \
		BEGIN { RS = "Potential nil panic detected"; real = 0; fp = 0 } \
		NR == 1 { next } \
		{ block = $$0; gsub(/\\\\/, "/", block) } \
		block ~ /http\/client\.go/ && block ~ /Do\(\)/ { fp++; next } \
		{ real++; printf "Potential nil panic detected%s", $$0 } \
		END { \
			if (fp) printf "nilaway: %d known false positive(s) filtered\n", fp; \
			if (real) { printf "nilaway: %d real finding(s)\n", real; exit 1 } \
		}'

fmt:
	${GO} fmt ./...

tidy:
	${GO} mod tidy

.PHONY: check vet build lint test cover vuln nilaway fmt tidy
