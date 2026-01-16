# All these commands must run from repository root.

vlgen:
	APP_NAME=vlgen $(MAKE) app-local

vlgen-race:
	APP_NAME=vlgen RACE=-race $(MAKE) app-local

vlgen-prod:
	APP_NAME=vlgen $(MAKE) app-via-docker

vlgen-pure-prod:
	APP_NAME=vlgen $(MAKE) app-via-docker-pure

vlgen-linux-amd64-prod:
	APP_NAME=vlgen $(MAKE) app-via-docker-linux-amd64

vlgen-linux-arm-prod:
	APP_NAME=vlgen $(MAKE) app-via-docker-linux-arm

vlgen-linux-arm64-prod:
	APP_NAME=vlgen $(MAKE) app-via-docker-linux-arm64

vlgen-linux-ppc64le-prod:
	APP_NAME=vlgen $(MAKE) app-via-docker-linux-ppc64le

vlgen-linux-386-prod:
	APP_NAME=vlgen $(MAKE) app-via-docker-linux-386

vlgen-darwin-amd64-prod:
	APP_NAME=vlgen $(MAKE) app-via-docker-darwin-amd64

vlgen-darwin-arm64-prod:
	APP_NAME=vlgen $(MAKE) app-via-docker-darwin-arm64

vlgen-freebsd-amd64-prod:
	APP_NAME=vlgen $(MAKE) app-via-docker-freebsd-amd64

vlgen-openbsd-amd64-prod:
	APP_NAME=vlgen $(MAKE) app-via-docker-openbsd-amd64

vlgen-windows-amd64-prod:
	APP_NAME=vlgen $(MAKE) app-via-docker-windows-amd64

vlgen-linux-amd64:
	APP_NAME=vlgen CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(MAKE) app-local-goos-goarch

vlgen-linux-arm:
	APP_NAME=vlgen CGO_ENABLED=0 GOOS=linux GOARCH=arm $(MAKE) app-local-goos-goarch

vlgen-linux-arm64:
	APP_NAME=vlgen CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(MAKE) app-local-goos-goarch

vlgen-linux-ppc64le:
	APP_NAME=vlgen CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le $(MAKE) app-local-goos-goarch

vlgen-linux-s390x:
	APP_NAME=vlgen CGO_ENABLED=0 GOOS=linux GOARCH=s390x $(MAKE) app-local-goos-goarch

vlgen-linux-loong64:
	APP_NAME=vlgen CGO_ENABLED=0 GOOS=linux GOARCH=loong64 $(MAKE) app-local-goos-goarch

vlgen-linux-386:
	APP_NAME=vlgen CGO_ENABLED=0 GOOS=linux GOARCH=386 $(MAKE) app-local-goos-goarch

vlgen-darwin-amd64:
	APP_NAME=vlgen CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(MAKE) app-local-goos-goarch

vlgen-darwin-arm64:
	APP_NAME=vlgen CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(MAKE) app-local-goos-goarch

vlgen-freebsd-amd64:
	APP_NAME=vlgen CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 $(MAKE) app-local-goos-goarch

vlgen-openbsd-amd64:
	APP_NAME=vlgen CGO_ENABLED=0 GOOS=openbsd GOARCH=amd64 $(MAKE) app-local-goos-goarch

vlgen-windows-amd64:
	GOARCH=amd64 APP_NAME=vlgen $(MAKE) app-local-windows-goarch

vlgen-pure:
	APP_NAME=vlgen $(MAKE) app-local-pure

run-vlgen:
	./bin/vlgen