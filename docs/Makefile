UID := "$(shell id -u)"
PLATFORM := $(shell uname -m)
DOCKER_PLATFORM := "linux/$(if $(findstring $(PLATFORM),arm64),arm64,amd64)"

define FLAGS_HEADER
---
build:
  list: never
  publishResources: false
  render: never
sitemap:
  disable: true
---
<!-- The file has to be manually updated during feature work in PR, "TAG=... make docs-update-flags" command could be used periodically to ensure the flags in sync with the given release TAG. -->
```shellhelp
endef
export FLAGS_HEADER

# These commands must be run from the VictoriaTraces repository root

docs-image: docs-repo-checkout
	docker build \
		-t vmdocs-docker-package \
		--build-arg UID=$(UID) \
		--platform $(DOCKER_PLATFORM) \
		vmdocs

docs-repo-checkout:
	if [ ! -d vmdocs ]; then \
		git clone --depth 1 git@github.com:VictoriaMetrics/vmdocs vmdocs; \
	fi; \
	cd vmdocs && git checkout origin/main && rm -rf content/victoriatraces && git checkout ./content/victoriatraces && git pull && cd ..

docs-debug: docs-image
	docker run \
		--rm \
		--name vmdocs-docker-container \
		--platform $(DOCKER_PLATFORM) \
		-p 1313:1313 \
		-v ./docs/victoriatraces:/opt/docs/content/victoriatraces \
			vmdocs-docker-package

docs-update-version:
	find docs/victoriatraces/ -name '*.md' -exec sed -i 's/{{% available_from "#" %}}/{{% available_from "$(TAG)" %}}/g' {} \;

# Converts images at docs folder to webp format
# See https://docs.victoriametrics.com/victoriametrics/single-server-victoriametrics/#images-in-documentation
docs-images-to-webp: docs-image
	docker run \
		--rm \
		--platform $(DOCKER_PLATFORM) \
		--entrypoint /usr/bin/find \
		--name vmdocs-docker-container \
		-v ./docs/victoriatraces:/opt/docs/content/victoriatraces vmdocs-docker-package \
			content \
				-regex ".*\.\(png\|jpg\|jpeg\)" \
				-exec sh -c 'cwebp -preset drawing -m 6 -o $$(echo {} | cut -f-1 -d.).webp {} && rm -rf {}' {} \;

docs-check-links: docs-image
	rm -rf $(PWD)/docs/public
	docker run \
		--rm \
		--platform $(DOCKER_PLATFORM) \
		-v $(PWD)/docs/victoriatraces:/opt/docs/content/victoriatraces \
		-v $(PWD)/docs/public:/opt/docs/public \
		--entrypoint /bin/sh \
		vmdocs-docker-package \
			-c "yarn install && hugo --minify && yarn run check-links" || exit_code=$$?; \
			rm -rf $(PWD)/docs/public; \
			exit $$exit_code

docs-update-victoria-traces-flags:
	# ---- victoria-traces
	#(cd /tmp/vt-enterprise && make victoria-traces)
	(cd /tmp/vt-opensource && make victoria-traces)
	#(cd /tmp/vt-enterprise && ./bin/victoria-traces -help 2>&1) > /tmp/vt-enterprise/victoria_traces_enterprise_flags_tmp.md
	(cd /tmp/vt-opensource && ./bin/victoria-traces -help 2>&1) > /tmp/vt-opensource/victoria_traces_common_flags_tmp.md

	echo "$$FLAGS_HEADER" > docs/victoriatraces/victoria_traces_common_flags.md
	cat /tmp/vt-opensource/victoria_traces_common_flags_tmp.md >> docs/victoriatraces/victoria_traces_common_flags.md
	printf -- '```\n' >> docs/victoriatraces/victoria_traces_common_flags.md

	#echo "$$FLAGS_HEADER" > docs/victoriatraces/victoria_traces_enterprise_flags.md
	#$(call extract_enterprise_flags,/tmp/vt-opensource/victoria_traces_common_flags_tmp.md,/tmp/vt-enterprise/victoria_traces_enterprise_flags_tmp.md,docs/victoriatraces/victoria_traces_enterprise_flags.md)
	#printf -- '```\n' >> docs/victoriatraces/victoria_traces_enterprise_flags.md

	# replace tabs in output with one space
	sed -i.bak 's/\t/ /g' docs/victoriatraces/victoria_traces_common_flags.md
	#sed -i.bak 's/\t/ /g' docs/victoriatraces/victoria_traces_enterprise_flags.md

	# adjust flags with dynamic default values
	sed -i.bak '/Default number of parallel data readers to use for executing every query/ s/(default [0-9]\+)/(default 2x CPU cores)/' docs/victoriatraces/victoria_traces_common_flags.md
	sed -i.bak '/The maximum number of concurrent insert requests/ s/(default [0-9]\+)/(default 2x CPU cores)/' docs/victoriatraces/victoria_traces_common_flags.md
	sed -i.bak '/The maximum number of concurrent search requests\./ s/(default [0-9]\+)/(default vtselect.getDefaultMaxConcurrentRequests())/' docs/victoriatraces/victoria_traces_common_flags.md
	sed -i.bak '/The maximum number of concurrent goroutines to work with files;/ s/(default [0-9]\+)/(default 16x CPU cores, default capped at 256)/' docs/victoriatraces/victoria_traces_common_flags.md
	rm -f docs/victoriatraces/victoria_traces_common_flags.md.bak
	#rm -f docs/victoriatraces/victoria_traces_enterprise_flags.md.bak

# docs-update-flags updates flags in the documentation using the actual binaries compiled
# from the given TAG.
# The command also normalizes the output a bit.
#
# The command does not replace the need to manually update flags in the documentation when
# new flags are added or existing flags are updated. It just helps to keep the documentation
# in sync with the actual binaries.
#
# It can be run from any branch.
# All work happens inside temporary directories under /tmp.
# The script checks out the required TAG, builds the binaries, and updates the documentation.
# The current Git repository is not touched.
docs-update-flags:
	# Note for MacOS users:
	# You need to install gnu versions of sed and awk in order for in-place editing to work
	# Install using: brew install gnu-sed gawk
	# Add tools to PATH see how in `brew info gnu-sed` and `brew info gawk`

	#rm -rf /tmp/vt-enterprise
	#git worktree remove /tmp/vt-enterprise || true
	#git worktree add /tmp/vt-enterprise $(TAG)-enterprise

	rm -rf /tmp/vt-opensource
	git worktree remove /tmp/vt-opensource || true
	git worktree add /tmp/vt-opensource $(TAG)

	make docs-update-victoria-traces-flags
