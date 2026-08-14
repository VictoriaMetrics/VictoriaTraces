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

GOLANGCI_LINT_VERSION := 2.9.0

.PHONY: $(MAKECMDGOALS)

include app/*/Makefile
include codespell/Makefile
include docs/Makefile
include deployment/*/Makefile
include dashboards/Makefile
include package/release/Makefile

all: \
	victoria-traces-prod \
	vtagent-prod

clean:
	rm -rf bin/*

publish: \
	publish-victoria-traces \
	publish-vtagent

vtutils: \
	vlagent

vtutils-pure: \
	vtagent-pure

vtutils-linux-amd64: \
	vtagent-linux-amd64

vtutils-linux-arm64: \
	vtagent-linux-arm64

vtutils-linux-arm: \
	vtagent-linux-arm

vtutils-linux-386: \
	vtagent-linux-386

vtutils-linux-ppc64le: \
	vtagent-linux-ppc64le

vtutils-linux-s390x: \
	vlagent-linux-s390x

vtutils-darwin-amd64: \
	vtagent-darwin-amd64

vtutils-darwin-arm64: \
	vtagent-darwin-arm64

vtutils-freebsd-amd64: \
	vtagent-freebsd-amd64

vtutils-openbsd-amd64: \
	vtagent-openbsd-amd64

vtutils-windows-amd64: \
	vtagent-windows-amd64

crossbuild:
	$(MAKE_PARALLEL) victoria-traces-crossbuild vtutils-crossbuild

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

vtutils-crossbuild: \
	vtutils-linux-386 \
	vtutils-linux-amd64 \
	vtutils-linux-arm64 \
	vtutils-linux-arm \
	vtutils-linux-ppc64le \
	vtutils-linux-s390x \
	vtutils-darwin-amd64 \
	vtutils-darwin-arm64 \
	vtutils-freebsd-amd64 \
	vtutils-openbsd-amd64 \
	vtutils-windows-amd64

publish-final-images:
	PKG_TAG=$(TAG) APP_NAME=victoria-traces $(MAKE) publish-via-docker-from-rc && \
	PKG_TAG=$(TAG) $(MAKE) publish-latest

publish-latest:
	PKG_TAG=$(TAG) APP_NAME=victoria-traces $(MAKE) publish-via-docker-latest
	PKG_TAG=$(TAG) APP_NAME=vtagent $(MAKE) publish-via-docker-latest

publish-release:
	rm -rf bin/*
	git checkout $(TAG) && $(MAKE) release && $(MAKE) publish

release: \
	release-victoria-traces \
	release-vtutils

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

release-vtutils: \
	release-vtutils-linux-386 \
	release-vtutils-linux-amd64 \
	release-vtutils-linux-arm64 \
	release-vtutils-linux-arm \
	release-vtutils-linux-s390x \
	release-vtutils-darwin-amd64 \
	release-vtutils-darwin-arm64 \
	release-vtutils-freebsd-amd64 \
	release-vtutils-openbsd-amd64 \
	release-vtutils-windows-amd64

release-vtutils-linux-386:
	GOOS=linux GOARCH=386 $(MAKE) release-vtutils-goos-goarch

release-vtutils-linux-amd64:
	GOOS=linux GOARCH=amd64 $(MAKE) release-vtutils-goos-goarch

release-vtutils-linux-arm64:
	GOOS=linux GOARCH=arm64 $(MAKE) release-vtutils-goos-goarch

release-vtutils-linux-arm:
	GOOS=linux GOARCH=arm $(MAKE) release-vtutils-goos-goarch

release-vtutils-linux-s390x:
	GOOS=linux GOARCH=s390x $(MAKE) release-vtutils-goos-goarch

release-vtutils-darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(MAKE) release-vtutils-goos-goarch

release-vtutils-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(MAKE) release-vtutils-goos-goarch

release-vtutils-freebsd-amd64:
	GOOS=freebsd GOARCH=amd64 $(MAKE) release-vtutils-goos-goarch

release-vtutils-openbsd-amd64:
	GOOS=openbsd GOARCH=amd64 $(MAKE) release-vtutils-goos-goarch

release-vtutils-windows-amd64:
	GOARCH=amd64 $(MAKE) release-vtutils-windows-goarch

release-vtutils-goos-goarch: \
	vtagent-$(GOOS)-$(GOARCH)-prod
	cd bin && \
		tar $(TAR_OWNERSHIP) --transform="flags=r;s|-$(GOOS)-$(GOARCH)||" -czf vtutils-$(GOOS)-$(GOARCH)-$(PKG_TAG).tar.gz \
			vtagent-$(GOOS)-$(GOARCH)-prod \
		&& sha256sum vtutils-$(GOOS)-$(GOARCH)-$(PKG_TAG).tar.gz \
			vtagent-$(GOOS)-$(GOARCH)-prod \
			| sed s/-$(GOOS)-$(GOARCH)-prod/-prod/ > vtutils-$(GOOS)-$(GOARCH)-$(PKG_TAG)_checksums.txt
	cd bin && rm -rf \
		vtagent-$(GOOS)-$(GOARCH)-prod

release-vtutils-windows-goarch: \
	vtagent-windows-$(GOARCH)-prod
	cd bin && \
		zip vtutils-windows-$(GOARCH)-$(PKG_TAG).zip \
			vtagent-windows-$(GOARCH)-prod.exe \
		&& sha256sum vtutils-windows-$(GOARCH)-$(PKG_TAG).zip \
			vtagent-windows-$(GOARCH)-prod.exe \
			> vtutils-windows-$(GOARCH)-$(PKG_TAG)_checksums.txt
	cd bin && rm -rf \
		vtagent-windows-$(GOARCH)-prod.exe


pprof-cpu:
	go tool pprof -trim_path=github.com/VictoriaMetrics/VictoriaTraces@ $(PPROF_FILE)

fmt:
	gofmt -l -w -s ./lib
	gofmt -l -w -s ./app
	gofmt -l -w -s ./apptest

vet:
	go vet -tags 'synctest' ./lib/...
	go vet ./app/...
	go vet ./apptest/...

check-all: fmt vet golangci-lint govulncheck

clean-checkers: remove-golangci-lint remove-govulncheck

test:
	go test -tags 'synctest' ./lib/... ./app/...

test-race:
	go test -tags 'synctest' -race ./lib/... ./app/...

test-pure:
	CGO_ENABLED=0 go test -tags 'synctest' ./lib/... ./app/...

test-full:
	go test -tags 'synctest' -coverprofile=coverage.txt -covermode=atomic ./lib/... ./app/...

test-full-386:
	GOARCH=386 go test -tags 'synctest' -coverprofile=coverage.txt -covermode=atomic ./lib/... ./app/...

integration-test:
	$(MAKE) apptest

apptest:
	$(MAKE) victoria-traces-race vtagent-race
	go test ./apptest/...

benchmark:
	go test -tags 'synctest' -bench=. ./lib/...
	go test -bench=. ./app/...

benchmark-pure:
	CGO_ENABLED=0 go test -run=NO_TESTS -bench=. ./lib/...
	CGO_ENABLED=0 go test -run=NO_TESTS -bench=. ./app/...

vendor-update:
	go get -u ./lib/...
	go get -u ./app/...
	go mod tidy -compat=1.26
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
	golangci-lint run --build-tags 'synctest'

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
