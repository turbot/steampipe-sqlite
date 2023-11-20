build:
	go run generate/generator.go templates . $(plugin_alias) $(plugin_github_url)
	go mod tidy
	make -f out/Makefile build
