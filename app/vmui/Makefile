# All these commands must run from repository root.

vmui-package-base-image:
	docker build -t vmui-builder-image -f app/vmui/Dockerfile-build ./app/vmui

vmui-run-npm-command: vmui-package-base-image
	docker run --rm \
		--user $(shell id -u):$(shell id -g) \
		--mount type=bind,src="$(shell pwd)/app/vmui",dst=/build \
		-w /build/packages/vmui \
		--entrypoint=/bin/bash \
		vmui-builder-image -c "npm ci && $(NPM_COMMAND)"

vmui-build:
	NPM_COMMAND="npm run build" $(MAKE) vmui-run-npm-command

vmui-update: vmui-build
	rm -rf app/vtselect/vmui/* && mv app/vmui/packages/vmui/build/* app/vtselect/vmui && rm -rf app/vtselect/vmui/dashboards

vmui-install:
	NPM_COMMAND="true" $(MAKE) vmui-run-npm-command

vmui-lint:
	NPM_COMMAND="npm run lint" $(MAKE) vmui-run-npm-command

vmui-typecheck:
	NPM_COMMAND="npm run typecheck" $(MAKE) vmui-run-npm-command

vmui-test:
	NPM_COMMAND="npm run test" $(MAKE) vmui-run-npm-command