ARG:=
MAKEFILE_DIR:=$(dir $(abspath $(lastword $(MAKEFILE_LIST))))

GOTESTSUM_VERSION:=v1.13.0 # renovate: datasource=github-releases depName=gotestyourself/gotestsum
GOTESTREPORT_VERSION:=v0.9.3 # renovate: datasource=github-releases depName=vakenbolt/go-test-report

build/gotestsum:
	mkdir -p build
	GOBIN=$(MAKEFILE_DIR)/build go install gotest.tools/gotestsum@${GOTESTSUM_VERSION}

build/go-test-report:
	mkdir -p build
	GOBIN=$(MAKEFILE_DIR)/build go install github.com/vakenbolt/go-test-report@${GOTESTREPORT_VERSION}

# Run tests
# if you want to update snapshot, run `make test ARG=-update`
.PHONY: test
test: build/gotestsum build/go-test-report
	mkdir -p build/reports
		build/gotestsum \
		--post-run-command "make test-post-run" \
		--format standard-verbose \
		--jsonfile build/reports/test-reports.json \
		--junitfile build/reports/test-reports.xml \
		--  ./... -race -coverprofile=build/reports/coverage.out ${ARG}

.PHONY: test-post-run
test-post-run:
	go tool cover -html=build/reports/coverage.out -o build/reports/coverage.html
	cat build/reports/test-reports.json | build/go-test-report -o build/reports/test-reports.html