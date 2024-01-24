GOLANG_CROSS_VERSION  ?= v1.21.5

# Check if the 'plugin' variable is set
validate_plugin:
ifndef plugin
	$(error "The 'plugin' variable is missing. Usage: make build plugin=<plugin_name>")
endif

build: validate_plugin
	go run generate/generator.go templates . $(plugin) $(plugin_github_url)
	go mod tidy
	$(MAKE) -f out/Makefile build

.PHONY: release-dry-run
release-dry-run:
	@docker run \
		--rm \
		-e CGO_ENABLED=1 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/steampipe-sqlite \
		-w /go/src/steampipe-sqlite \
		ghcr.io/goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
		--clean --skip=validate --skip=publish --snapshot
