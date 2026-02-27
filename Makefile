ARG:=
MAKEFILE_DIR:=$(dir $(abspath $(lastword $(MAKEFILE_LIST))))

GOTESTSUM_VERSION:=v1.13.0 # renovate: datasource=github-releases depName=gotestyourself/gotestsum

build/gotestsum:
	mkdir -p build
	GOBIN=$(MAKEFILE_DIR)/build go install gotest.tools/gotestsum@${GOTESTSUM_VERSION}

# Run tests
# if you want to update snapshot, run `make test ARG=-update`
.PHONY: test
test: build/gotestsum
	mkdir -p build/reports
		build/gotestsum \
		--post-run-command "go tool cover -html=build/reports/coverage.out -o build/reports/coverage.html" \
		--format standard-verbose \
		--jsonfile build/reports/test-reports.json \
		--junitfile build/reports/test-reports.xml \
		--  ./... -race -coverprofile=build/reports/coverage.out ${ARG}