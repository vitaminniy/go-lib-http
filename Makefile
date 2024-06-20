all: help

help: ## prints this help
help:
	@IFS=$$'\n' ; \
	help_lines=(`fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##/:/'`); \
	printf "%-20s %s\n" "target" "help" ; \
	printf "%-20s %s\n" "------" "----" ; \
	for help_line in $${help_lines[@]}; do \
		IFS=$$':' ; \
		help_split=($$help_line) ; \
		help_command=`echo $${help_split[0]} | sed -e 's/^ *//' -e 's/ *$$//'` ; \
		help_info=`echo $${help_split[2]} | sed -e 's/^ *//' -e 's/ *$$//'` ; \
		printf '\033[36m'; \
		printf "%-20s %s" $$help_command ; \
		printf '\033[0m'; \
		printf "%s\n" $$help_info; \
	done

install: ## install everything
install: install-gen-client
.PHONY: install

install-gen-client: ## installs http client generator
	go install ./cmd/gen-client
.PHONY: install-gen-client

test: ## runs tests
test: test-unit
.PHONY: tests

test-unit: ## runs unit tests
test-unit:
	go test -race ./...
.PHONY: test-unit


EXAMPLES = $(wildcard examples/*)

generate-examples: ## generates examples
generate-examples: install-gen-client
	@for example in $(EXAMPLES); do \
		$(MAKE) -C $$example generate; \
	done
.PHONY: generate-examples

build-examples: ## builds examples
build-examples: generate-examples
	go build -v ./examples/...
.PHONY: build-examples
