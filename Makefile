BINDIR ?= ./bin
PREFIX ?= $(HOME)/.local/bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -X main.version=$(VERSION)

.PHONY: build test install uninstall

build:
	mkdir -p $(BINDIR)
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/canlc ./compiler

test:
	go test ./...

install: build
	mkdir -p $(PREFIX)
	cp $(BINDIR)/canlc $(PREFIX)/canlc
	@echo "installed canlc to $(PREFIX)/canlc"

uninstall:
	rm -f $(PREFIX)/canlc

# The archive is acquired explicitly; assembling and running never downloads Bun.
.PHONY: bundle
bundle:
	@test -n "$(BUN_ARCHIVE)" || (echo "BUN_ARCHIVE must name the pinned local zip" >&2; exit 2)
	go run ./tools/distbuild --archive "$(BUN_ARCHIVE)" --out "$(BUNDLE_OUT)" --version "$(VERSION)"

BUNDLE_OUT ?= ./dist/development
