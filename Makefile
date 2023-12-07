# Check if the 'plugin' variable is set
validate_plugin:
ifndef plugin
	$(error "The 'plugin' variable is missing. Usage: make build plugin=<plugin_name>")
endif

build: validate_plugin
	go run generate/generator.go templates . $(plugin) $(plugin_github_url)
	go mod tidy
	$(MAKE) -f out/Makefile build
