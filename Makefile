SWIFT_CACHE_FLAGS = --package-path apps/macos --cache-path "$(CURDIR)/.build/swiftpm-cache" --config-path "$(CURDIR)/.build/swiftpm-config" --security-path "$(CURDIR)/.build/swiftpm-security" --scratch-path "$(CURDIR)/.build/swiftpm-scratch" --manifest-cache local --disable-sandbox
SWIFT_ENV = CLANG_MODULE_CACHE_PATH="$(CURDIR)/.build/clang-module-cache"
GO_ENV = GOCACHE="$(CURDIR)/.build/go-cache"

.PHONY: bootstrap test test-go test-macos lint lint-go lint-swift licence-check fetch-research docs-check infra-check clean-workspace-check

bootstrap:
	go work sync
	mkdir -p ".build/clang-module-cache" ".build/swiftpm-cache" ".build/swiftpm-config" ".build/swiftpm-security" ".build/swiftpm-scratch"
	$(SWIFT_ENV) swift package resolve $(SWIFT_CACHE_FLAGS)
	./scripts/docs-check.sh

test: test-go test-macos docs-check

test-go:
	mkdir -p ".build/go-cache"
	$(GO_ENV) go test ./...

test-macos:
	mkdir -p ".build/clang-module-cache" ".build/swiftpm-cache" ".build/swiftpm-config" ".build/swiftpm-security" ".build/swiftpm-scratch"
	$(SWIFT_ENV) swift build $(SWIFT_CACHE_FLAGS)

lint: lint-go lint-swift docs-check

lint-go:
	@test -z "$$(gofmt -l $$(find . -path './.research-src' -prune -o -name '*.go' -print))"

lint-swift:
	@if command -v swiftlint >/dev/null 2>&1; then \
		swiftlint lint --strict; \
	else \
		echo "swiftlint not installed; skipping local SwiftLint run"; \
	fi

licence-check:
	./scripts/licence-check.sh

fetch-research:
	./scripts/fetch-research-sources.sh

docs-check:
	./scripts/docs-check.sh

infra-check:
	@if command -v tofu >/dev/null 2>&1; then \
		tofu -chdir=deploy/tofu/environments/dev init -backend=false && tofu -chdir=deploy/tofu/environments/dev validate; \
	elif command -v terraform >/dev/null 2>&1; then \
		terraform -chdir=deploy/tofu/environments/dev init -backend=false && terraform -chdir=deploy/tofu/environments/dev validate; \
	else \
		echo "OpenTofu/Terraform not installed; skipping local infra validation"; \
	fi

clean-workspace-check:
	./scripts/validate-clean-workspace.sh
