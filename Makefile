PKG_PREFIX := github.com/VictoriaMetrics/VictoriaTraces

MAKE_CONCURRENCY ?= $(shell getconf _NPROCESSORS_ONLN)
MAKE_PARALLEL := $(MAKE) -j $(MAKE_CONCURRENCY)
DATEINFO_TAG ?= $(shell date -u +'%Y%m%d-%H%M%S')
BUILDINFO_TAG ?= $(shell echo $$(git describe --long --all | tr '/' '-')$$( \
	      git diff-index --quiet HEAD -- || echo '-dirty-'$$(git diff-index -u HEAD | openssl sha1 | cut -d' ' -f2 | cut -c 1-8)))

PKG_TAG ?= $(shell git tag -l --points-at HEAD)
ifeq ($(PKG_TAG),)
PKG_TAG := $(BUILDINFO_TAG)
endif

GO_BUILDINFO = -X 'github.com/VictoriaMetrics/VictoriaMetrics/lib/buildinfo.Version=$(APP_NAME)-$(DATEINFO_TAG)-$(BUILDINFO_TAG)'
TAR_OWNERSHIP ?= --owner=1000 --group=1000

GOLANGCI_LINT_VERSION := 2.4.0

.PHONY: $(MAKECMDGOALS)

include app/*/Makefile
include codespell/Makefile
include docs/Makefile
include deployment/*/Makefile
include dashboards/Makefile
include package/release/Makefile

all: \
	victoria-traces-prod

clean:
	rm -rf bin/*

publish: \
	publish-victoria-traces

package: \
	package-victoria-traces

crossbuild:
	$(MAKE_PARALLEL) victoria-traces-crossbuild

victoria-traces-crossbuild: \
	victoria-traces-linux-386 \
	victoria-traces-linux-amd64 \
	victoria-traces-linux-arm64 \
	victoria-traces-linux-arm \
	victoria-traces-linux-ppc64le \
	victoria-traces-linux-s390x \
	victoria-traces-darwin-amd64 \
	victoria-traces-darwin-arm64 \
	victoria-traces-freebsd-amd64 \
	victoria-traces-openbsd-amd64 \
	victoria-traces-windows-amd64

publish-final-images:
	PKG_TAG=$(TAG) APP_NAME=victoria-traces $(MAKE) publish-via-docker-from-rc && \
	PKG_TAG=$(TAG) $(MAKE) publish-latest

publish-latest:
	PKG_TAG=$(TAG) APP_NAME=victoria-traces $(MAKE) publish-via-docker-latest

publish-release:
	rm -rf bin/*
	git checkout $(TAG) && $(MAKE) release && $(MAKE) publish

release: \
	release-victoria-traces

release-victoria-traces:
	$(MAKE_PARALLEL) release-victoria-traces-linux-386 \
		release-victoria-traces-linux-amd64 \
		release-victoria-traces-linux-arm \
		release-victoria-traces-linux-arm64 \
		release-victoria-traces-linux-s390x \
		release-victoria-traces-darwin-amd64 \
		release-victoria-traces-darwin-arm64 \
		release-victoria-traces-freebsd-amd64 \
		release-victoria-traces-openbsd-amd64 \
		release-victoria-traces-windows-amd64

release-victoria-traces-linux-386:
	GOOS=linux GOARCH=386 $(MAKE) release-victoria-traces-goos-goarch

release-victoria-traces-linux-amd64:
	GOOS=linux GOARCH=amd64 $(MAKE) release-victoria-traces-goos-goarch

release-victoria-traces-linux-arm:
	GOOS=linux GOARCH=arm $(MAKE) release-victoria-traces-goos-goarch

release-victoria-traces-linux-arm64:
	GOOS=linux GOARCH=arm64 $(MAKE) release-victoria-traces-goos-goarch

release-victoria-traces-linux-s390x:
	GOOS=linux GOARCH=s390x $(MAKE) release-victoria-traces-goos-goarch

release-victoria-traces-darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(MAKE) release-victoria-traces-goos-goarch

release-victoria-traces-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(MAKE) release-victoria-traces-goos-goarch

release-victoria-traces-freebsd-amd64:
	GOOS=freebsd GOARCH=amd64 $(MAKE) release-victoria-traces-goos-goarch

release-victoria-traces-openbsd-amd64:
	GOOS=openbsd GOARCH=amd64 $(MAKE) release-victoria-traces-goos-goarch

release-victoria-traces-windows-amd64:
	GOARCH=amd64 $(MAKE) release-victoria-traces-windows-goarch

release-victoria-traces-goos-goarch: victoria-traces-$(GOOS)-$(GOARCH)-prod
	cd bin && \
		tar $(TAR_OWNERSHIP) --transform="flags=r;s|-$(GOOS)-$(GOARCH)||" -czf victoria-traces-$(GOOS)-$(GOARCH)-$(PKG_TAG).tar.gz \
			victoria-traces-$(GOOS)-$(GOARCH)-prod \
		&& sha256sum victoria-traces-$(GOOS)-$(GOARCH)-$(PKG_TAG).tar.gz \
			victoria-traces-$(GOOS)-$(GOARCH)-prod \
			| sed s/-$(GOOS)-$(GOARCH)-prod/-prod/ > victoria-traces-$(GOOS)-$(GOARCH)-$(PKG_TAG)_checksums.txt
	cd bin && rm -rf victoria-traces-$(GOOS)-$(GOARCH)-prod

release-victoria-traces-windows-goarch: victoria-traces-windows-$(GOARCH)-prod
	cd bin && \
		zip victoria-traces-windows-$(GOARCH)-$(PKG_TAG).zip \
			victoria-traces-windows-$(GOARCH)-prod.exe \
		&& sha256sum victoria-traces-windows-$(GOARCH)-$(PKG_TAG).zip \
			victoria-traces-windows-$(GOARCH)-prod.exe \
			> victoria-traces-windows-$(GOARCH)-$(PKG_TAG)_checksums.txt
	cd bin && rm -rf \
		victoria-traces-windows-$(GOARCH)-prod.exe

pprof-cpu:
	go tool pprof -trim_path=github.com/VictoriaMetrics/VictoriaTraces@ $(PPROF_FILE)

fmt:
	gofmt -l -w -s ./lib
	gofmt -l -w -s ./app
	gofmt -l -w -s ./apptest

vet:
	GOEXPERIMENT=synctest go vet ./lib/...
	go vet ./app/...
	go vet ./apptest/...

check-all: fmt vet golangci-lint govulncheck

clean-checkers: remove-golangci-lint remove-govulncheck

test:
	GOEXPERIMENT=synctest go test ./lib/... ./app/...

test-race:
	GOEXPERIMENT=synctest go test -race ./lib/... ./app/...

test-pure:
	GOEXPERIMENT=synctest CGO_ENABLED=0 go test ./lib/... ./app/...

test-full:
	GOEXPERIMENT=synctest go test -coverprofile=coverage.txt -covermode=atomic ./lib/... ./app/...

test-full-386:
	GOEXPERIMENT=synctest GOARCH=386 go test -coverprofile=coverage.txt -covermode=atomic ./lib/... ./app/...

integration-test:
	$(MAKE) apptest

apptest:
	$(MAKE) victoria-traces
	go test ./apptest/...

benchmark:
	GOEXPERIMENT=synctest go test -bench=. ./lib/...
	go test -bench=. ./app/...

benchmark-pure:
	GOEXPERIMENT=synctest CGO_ENABLED=0 go test -bench=. ./lib/...
	CGO_ENABLED=0 go test -bench=. ./app/...

vendor-update:
	go get -u ./lib/...
	go get -u ./app/...
	go mod tidy -compat=1.25
	go mod vendor

app-local:
	CGO_ENABLED=1 go build $(RACE) -ldflags "$(GO_BUILDINFO)" -o bin/$(APP_NAME)$(RACE) $(PKG_PREFIX)/app/$(APP_NAME)

app-local-pure:
	CGO_ENABLED=0 go build $(RACE) -ldflags "$(GO_BUILDINFO)" -o bin/$(APP_NAME)-pure$(RACE) $(PKG_PREFIX)/app/$(APP_NAME)

app-local-goos-goarch:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(RACE) -ldflags "$(GO_BUILDINFO)" -o bin/$(APP_NAME)-$(GOOS)-$(GOARCH)$(RACE) $(PKG_PREFIX)/app/$(APP_NAME)

app-local-windows-goarch:
	CGO_ENABLED=0 GOOS=windows GOARCH=$(GOARCH) go build $(RACE) -ldflags "$(GO_BUILDINFO)" -o bin/$(APP_NAME)-windows-$(GOARCH)$(RACE).exe $(PKG_PREFIX)/app/$(APP_NAME)

quicktemplate-gen: install-qtc
	qtc

install-qtc:
	which qtc || go install github.com/valyala/quicktemplate/qtc@latest

golangci-lint: install-golangci-lint
	GOEXPERIMENT=synctest golangci-lint run

install-golangci-lint:
	which golangci-lint && (golangci-lint --version | grep -q $(GOLANGCI_LINT_VERSION)) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v$(GOLANGCI_LINT_VERSION)

remove-golangci-lint:
	rm -rf `which golangci-lint`

govulncheck: install-govulncheck
	govulncheck ./...

install-govulncheck:
	which govulncheck || go install golang.org/x/vuln/cmd/govulncheck@latest

remove-govulncheck:
	rm -rf `which govulncheck`

install-wwhrd:
	which wwhrd || go install github.com/frapposelli/wwhrd@latest

check-licenses: install-wwhrd
	wwhrd check -f .wwhrd.yml
