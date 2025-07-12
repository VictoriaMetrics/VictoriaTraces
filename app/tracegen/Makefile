# All these commands must run from repository root.

tracegen:
	APP_NAME=tracegen $(MAKE) app-local

tracegen-race:
	APP_NAME=tracegen RACE=-race $(MAKE) app-local

tracegen-prod:
	APP_NAME=tracegen $(MAKE) app-via-docker

tracegen-pure-prod:
	APP_NAME=tracegen $(MAKE) app-via-docker-pure

tracegen-linux-amd64-prod:
	APP_NAME=tracegen $(MAKE) app-via-docker-linux-amd64

tracegen-linux-arm-prod:
	APP_NAME=tracegen $(MAKE) app-via-docker-linux-arm

tracegen-linux-arm64-prod:
	APP_NAME=tracegen $(MAKE) app-via-docker-linux-arm64

tracegen-linux-ppc64le-prod:
	APP_NAME=tracegen $(MAKE) app-via-docker-linux-ppc64le

tracegen-linux-386-prod:
	APP_NAME=tracegen $(MAKE) app-via-docker-linux-386

tracegen-darwin-amd64-prod:
	APP_NAME=tracegen $(MAKE) app-via-docker-darwin-amd64

tracegen-darwin-arm64-prod:
	APP_NAME=tracegen $(MAKE) app-via-docker-darwin-arm64

tracegen-freebsd-amd64-prod:
	APP_NAME=tracegen $(MAKE) app-via-docker-freebsd-amd64

tracegen-openbsd-amd64-prod:
	APP_NAME=tracegen $(MAKE) app-via-docker-openbsd-amd64

tracegen-windows-amd64-prod:
	APP_NAME=tracegen $(MAKE) app-via-docker-windows-amd64

tracegen-linux-amd64:
	APP_NAME=tracegen CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(MAKE) app-local-goos-goarch

tracegen-linux-arm:
	APP_NAME=tracegen CGO_ENABLED=0 GOOS=linux GOARCH=arm $(MAKE) app-local-goos-goarch

tracegen-linux-arm64:
	APP_NAME=tracegen CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(MAKE) app-local-goos-goarch

tracegen-linux-ppc64le:
	APP_NAME=tracegen CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le $(MAKE) app-local-goos-goarch

tracegen-linux-s390x:
	APP_NAME=tracegen CGO_ENABLED=0 GOOS=linux GOARCH=s390x $(MAKE) app-local-goos-goarch

tracegen-linux-loong64:
	APP_NAME=tracegen CGO_ENABLED=0 GOOS=linux GOARCH=loong64 $(MAKE) app-local-goos-goarch

tracegen-linux-386:
	APP_NAME=tracegen CGO_ENABLED=0 GOOS=linux GOARCH=386 $(MAKE) app-local-goos-goarch

tracegen-darwin-amd64:
	APP_NAME=tracegen CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(MAKE) app-local-goos-goarch

tracegen-darwin-arm64:
	APP_NAME=tracegen CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(MAKE) app-local-goos-goarch

tracegen-freebsd-amd64:
	APP_NAME=tracegen CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 $(MAKE) app-local-goos-goarch

tracegen-openbsd-amd64:
	APP_NAME=tracegen CGO_ENABLED=0 GOOS=openbsd GOARCH=amd64 $(MAKE) app-local-goos-goarch

tracegen-windows-amd64:
	GOARCH=amd64 APP_NAME=tracegen $(MAKE) app-local-windows-goarch

tracegen-pure:
	APP_NAME=tracegen $(MAKE) app-local-pure
