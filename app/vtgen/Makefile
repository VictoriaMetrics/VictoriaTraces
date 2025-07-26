# All these commands must run from repository root.

vtgen:
	APP_NAME=vtgen $(MAKE) app-local

vtgen-race:
	APP_NAME=vtgen RACE=-race $(MAKE) app-local

vtgen-prod:
	APP_NAME=vtgen $(MAKE) app-via-docker

vtgen-pure-prod:
	APP_NAME=vtgen $(MAKE) app-via-docker-pure

vtgen-linux-amd64-prod:
	APP_NAME=vtgen $(MAKE) app-via-docker-linux-amd64

vtgen-linux-arm-prod:
	APP_NAME=vtgen $(MAKE) app-via-docker-linux-arm

vtgen-linux-arm64-prod:
	APP_NAME=vtgen $(MAKE) app-via-docker-linux-arm64

vtgen-linux-ppc64le-prod:
	APP_NAME=vtgen $(MAKE) app-via-docker-linux-ppc64le

vtgen-linux-386-prod:
	APP_NAME=vtgen $(MAKE) app-via-docker-linux-386

vtgen-darwin-amd64-prod:
	APP_NAME=vtgen $(MAKE) app-via-docker-darwin-amd64

vtgen-darwin-arm64-prod:
	APP_NAME=vtgen $(MAKE) app-via-docker-darwin-arm64

vtgen-freebsd-amd64-prod:
	APP_NAME=vtgen $(MAKE) app-via-docker-freebsd-amd64

vtgen-openbsd-amd64-prod:
	APP_NAME=vtgen $(MAKE) app-via-docker-openbsd-amd64

vtgen-windows-amd64-prod:
	APP_NAME=vtgen $(MAKE) app-via-docker-windows-amd64

vtgen-linux-amd64:
	APP_NAME=vtgen CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(MAKE) app-local-goos-goarch

vtgen-linux-arm:
	APP_NAME=vtgen CGO_ENABLED=0 GOOS=linux GOARCH=arm $(MAKE) app-local-goos-goarch

vtgen-linux-arm64:
	APP_NAME=vtgen CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(MAKE) app-local-goos-goarch

vtgen-linux-ppc64le:
	APP_NAME=vtgen CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le $(MAKE) app-local-goos-goarch

vtgen-linux-s390x:
	APP_NAME=vtgen CGO_ENABLED=0 GOOS=linux GOARCH=s390x $(MAKE) app-local-goos-goarch

vtgen-linux-loong64:
	APP_NAME=vtgen CGO_ENABLED=0 GOOS=linux GOARCH=loong64 $(MAKE) app-local-goos-goarch

vtgen-linux-386:
	APP_NAME=vtgen CGO_ENABLED=0 GOOS=linux GOARCH=386 $(MAKE) app-local-goos-goarch

vtgen-darwin-amd64:
	APP_NAME=vtgen CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(MAKE) app-local-goos-goarch

vtgen-darwin-arm64:
	APP_NAME=vtgen CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(MAKE) app-local-goos-goarch

vtgen-freebsd-amd64:
	APP_NAME=vtgen CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 $(MAKE) app-local-goos-goarch

vtgen-openbsd-amd64:
	APP_NAME=vtgen CGO_ENABLED=0 GOOS=openbsd GOARCH=amd64 $(MAKE) app-local-goos-goarch

vtgen-windows-amd64:
	GOARCH=amd64 APP_NAME=vtgen $(MAKE) app-local-windows-goarch

vtgen-pure:
	APP_NAME=vtgen $(MAKE) app-local-pure

run-vtgen:
	./bin/vtgen