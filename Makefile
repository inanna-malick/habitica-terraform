.PHONY: build install test test-unit test-coverage test-clean test-verbose fmt vet clean fetch-tags fetch-habits fetch-dailies

# Build the provider binary
build:
	go build -o terraform-provider-habitica

# Format and vet
fmt:
	go fmt ./...

vet:
	go vet ./...

# Build + format + vet
all: fmt vet build

# Clean build artifacts
clean:
	rm -f terraform-provider-habitica

# Fetch existing resources from Habitica API (requires env vars set)
fetch-tags:
	@curl -s -H "x-api-user: $$HABITICA_USER_ID" \
		-H "x-api-key: $$HABITICA_API_TOKEN" \
		-H "x-client: $$HABITICA_CLIENT_AUTHOR_ID-TerraformHabitica" \
		https://habitica.com/api/v3/tags | jq '.data[] | {id, name}'

fetch-habits:
	@curl -s -H "x-api-user: $$HABITICA_USER_ID" \
		-H "x-api-key: $$HABITICA_API_TOKEN" \
		-H "x-client: $$HABITICA_CLIENT_AUTHOR_ID-TerraformHabitica" \
		"https://habitica.com/api/v3/tasks/user?type=habits" | jq '.data[] | {id, text, up, down}'

fetch-dailies:
	@curl -s -H "x-api-user: $$HABITICA_USER_ID" \
		-H "x-api-key: $$HABITICA_API_TOKEN" \
		-H "x-client: $$HABITICA_CLIENT_AUTHOR_ID-TerraformHabitica" \
		"https://habitica.com/api/v3/tasks/user?type=dailys" | jq '.data[] | {id, text, frequency}'

# Generate terraform import config with tag references (Python)
generate-imports:
	@rm -f examples/import/generated.tf
	python3 scripts/generate_imports.py

# Generate terraform import config (jq version, no tag linking)
generate-imports-jq:
	@echo 'terraform {' > examples/import/generated.tf
	@echo '  required_providers {' >> examples/import/generated.tf
	@echo '    habitica = { source = "registry.terraform.io/inannamalick/habitica" }' >> examples/import/generated.tf
	@echo '  }' >> examples/import/generated.tf
	@echo '}' >> examples/import/generated.tf
	@echo '' >> examples/import/generated.tf
	@echo 'provider "habitica" {}' >> examples/import/generated.tf
	@echo '' >> examples/import/generated.tf
	@echo '# === TAGS ===' >> examples/import/generated.tf
	@curl -s -H "x-api-user: $$HABITICA_USER_ID" \
		-H "x-api-key: $$HABITICA_API_TOKEN" \
		-H "x-client: $$HABITICA_CLIENT_AUTHOR_ID-TerraformHabitica" \
		https://habitica.com/api/v3/tags | jq -r '.data[] | "import {\n  to = habitica_tag.\(.name | gsub("[^a-zA-Z0-9]"; "_") | ascii_downcase)\n  id = \"\(.id)\"\n}\n\nresource \"habitica_tag\" \"\(.name | gsub("[^a-zA-Z0-9]"; "_") | ascii_downcase)\" {\n  name = \"\(.name)\"\n}\n"' >> examples/import/generated.tf
	@echo '# === HABITS ===' >> examples/import/generated.tf
	@curl -s -H "x-api-user: $$HABITICA_USER_ID" \
		-H "x-api-key: $$HABITICA_API_TOKEN" \
		-H "x-client: $$HABITICA_CLIENT_AUTHOR_ID-TerraformHabitica" \
		"https://habitica.com/api/v3/tasks/user?type=habits" | jq -r '.data[] | "import {\n  to = habitica_habit.\(.text | gsub("[^a-zA-Z0-9]"; "_") | ascii_downcase | .[0:50])\n  id = \"\(.id)\"\n}\n\nresource \"habitica_habit\" \"\(.text | gsub("[^a-zA-Z0-9]"; "_") | ascii_downcase | .[0:50])\" {\n  text     = \"\(.text | gsub("\""; "\\\""))\"\n  up       = \(.up)\n  down     = \(.down)\n  priority = \(.priority)\n}\n"' >> examples/import/generated.tf
	@echo '# === DAILIES ===' >> examples/import/generated.tf
	@curl -s -H "x-api-user: $$HABITICA_USER_ID" \
		-H "x-api-key: $$HABITICA_API_TOKEN" \
		-H "x-client: $$HABITICA_CLIENT_AUTHOR_ID-TerraformHabitica" \
		"https://habitica.com/api/v3/tasks/user?type=dailys" | jq -r '.data[] | "import {\n  to = habitica_daily.\(.text | gsub("[^a-zA-Z0-9]"; "_") | ascii_downcase | .[0:50])\n  id = \"\(.id)\"\n}\n\nresource \"habitica_daily\" \"\(.text | gsub("[^a-zA-Z0-9]"; "_") | ascii_downcase | .[0:50])\" {\n  text      = \"\(.text | gsub("\""; "\\\""))\"\n  frequency = \"\(.frequency)\"\n  every_x   = \(.everyX)\n  priority  = \(.priority)\n}\n"' >> examples/import/generated.tf
	@echo "Generated examples/import/generated.tf"

# Run terraform plan in examples/import (after setting up .terraformrc)
plan:
	cd examples/import && terraform plan

# Run terraform apply
apply:
	cd examples/import && terraform apply

# Test targets
test:
	go test -v -race -coverprofile=coverage.out ./...

test-unit:
	go test -v -race -short ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-verbose:
	go test -v -race -coverprofile=coverage.out ./... -count=1

test-clean:
	go clean -testcache
	rm -f coverage.out coverage.html
