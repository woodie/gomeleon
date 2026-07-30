.PHONY: build install lint test check

build:
	go build -o gomeleon

install: build
	mv gomeleon ~/go/bin/

# lint and test are always verbose. check is terse (silent on pass, full
# log on any failure) -- matching every other lint/test/check split in
# this account (see gorderly's/xctidy's own Makefiles).

lint:
	golangci-lint run

# Verbose on purpose -- Ginkgo's own -v output, already RSpec-documentation-
# style by nature (this repo's own reason for existing is reformatting this
# exact output for consumers who only have plain `go test`, not Ginkgo).
test:
	ginkgo -v ./...

# Terser than `test` on purpose: Ginkgo's default (non -v) run has no dot
# mode of its own here -- this just suppresses output on success and dumps
# the full log on failure, guaranteeing errors are never hidden.
check: lint
	@LOG=$$(mktemp); \
	if ginkgo ./... > "$$LOG" 2>&1; then \
		echo "PASS"; \
	else \
		cat "$$LOG"; \
		rm -f "$$LOG"; \
		exit 1; \
	fi; \
	rm -f "$$LOG"
