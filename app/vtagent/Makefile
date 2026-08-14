# All these commands must run from repository root.

vtagent:
	APP_NAME=vtagent $(MAKE) app-local

vtagent-race:
	APP_NAME=vtagent RACE=-race $(MAKE) app-local

vtagent-prod:
	APP_NAME=vtagent $(MAKE) app-via-docker

vtagent-pure-prod:
	APP_NAME=vtagent $(MAKE) app-via-docker-pure

vtagent-linux-amd64-prod:
	APP_NAME=vtagent $(MAKE) app-via-docker-linux-amd64

vtagent-linux-arm-prod:
	APP_NAME=vtagent $(MAKE) app-via-docker-linux-arm

vtagent-linux-arm64-prod:
	APP_NAME=vtagent $(MAKE) app-via-docker-linux-arm64

vtagent-linux-ppc64le-prod:
	APP_NAME=vtagent $(MAKE) app-via-docker-linux-ppc64le

vtagent-linux-s390x-prod:
	APP_NAME=vtagent $(MAKE) app-via-docker-linux-s390x

vtagent-linux-386-prod:
	APP_NAME=vtagent $(MAKE) app-via-docker-linux-386

vtagent-darwin-amd64-prod:
	APP_NAME=vtagent $(MAKE) app-via-docker-darwin-amd64

vtagent-darwin-arm64-prod:
	APP_NAME=vtagent $(MAKE) app-via-docker-darwin-arm64

vtagent-freebsd-amd64-prod:
	APP_NAME=vtagent $(MAKE) app-via-docker-freebsd-amd64

vtagent-openbsd-amd64-prod:
	APP_NAME=vtagent $(MAKE) app-via-docker-openbsd-amd64

vtagent-windows-amd64-prod:
	APP_NAME=vtagent $(MAKE) app-via-docker-windows-amd64

publish-vtagent:
	APP_NAME=vtagent $(MAKE) publish-via-docker

vtagent-linux-amd64:
	APP_NAME=vtagent CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(MAKE) app-local-goos-goarch

vtagent-linux-arm:
	APP_NAME=vtagent CGO_ENABLED=0 GOOS=linux GOARCH=arm $(MAKE) app-local-goos-goarch

vtagent-linux-arm64:
	APP_NAME=vtagent CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(MAKE) app-local-goos-goarch

vtagent-linux-ppc64le:
	APP_NAME=vtagent CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le $(MAKE) app-local-goos-goarch

vtagent-linux-s390x:
	APP_NAME=vtagent CGO_ENABLED=0 GOOS=linux GOARCH=s390x $(MAKE) app-local-goos-goarch

vtagent-linux-loong64:
	APP_NAME=vtagent CGO_ENABLED=0 GOOS=linux GOARCH=loong64 $(MAKE) app-local-goos-goarch

vtagent-linux-386:
	APP_NAME=vtagent CGO_ENABLED=0 GOOS=linux GOARCH=386 $(MAKE) app-local-goos-goarch

vtagent-darwin-amd64:
	APP_NAME=vtagent CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(MAKE) app-local-goos-goarch

vtagent-darwin-arm64:
	APP_NAME=vtagent CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(MAKE) app-local-goos-goarch

vtagent-freebsd-amd64:
	APP_NAME=vtagent CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 $(MAKE) app-local-goos-goarch

vtagent-openbsd-amd64:
	APP_NAME=vtagent CGO_ENABLED=0 GOOS=openbsd GOARCH=amd64 $(MAKE) app-local-goos-goarch

vtagent-windows-amd64:
	GOARCH=amd64 APP_NAME=vtagent $(MAKE) app-local-windows-goarch

vtagent-pure:
	APP_NAME=vtagent $(MAKE) app-local-pure
