# Check if the 'plugin' variable is set
validate_plugin:
ifndef plugin
	$(error "The 'plugin' variable is missing. Usage: make build plugin=<plugin_name>")
endif

build: validate_plugin

	# Create a new directory for the build process
	mkdir -p render

	# Copy the entire source tree, excluding .git directory, into the new directory
	rsync -a --exclude='.git' . render/ >/dev/null 2>&1

	# Change to the new directory to perform operations
	cd render && \
	go run generate/generator.go templates . $(plugin) $(plugin_github_url) && \
	go mod tidy && \
	$(MAKE) -f out/Makefile build

	# Copy the created binary from the render directory
	cp render/steampipe_sqlite_$(plugin).so .

	# Clean up the render directory
	rm -rf render

	# Note: The render directory will contain the full code tree with changes, 
	# binaries will be copied to the actual root directory, and then render will be deleted
