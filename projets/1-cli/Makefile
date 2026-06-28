# Makefile du Projet 1 — txtkit
#
# Cibles principales :
#   make build      compile le binaire local dans bin/txtkit
#   make test       go test -race ./...
#   make vet        go vet ./...
#   make dist       cross-compile pour linux/darwin/windows (amd64 + arm64)
#   make clean      supprime bin/ et dist/

BINARY  := txtkit
PKG     := example.com/txtkit/internal/cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X $(PKG).version=$(VERSION)

# Plateformes ciblées par « make dist » (cross-compilation pure Go, sans cgo).
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build test vet fmt dist clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

# Cross-compilation : on fixe GOOS/GOARCH par plateforme. CGO_ENABLED=0 garantit
# un binaire statique, sans dépendance à la libc de la machine cible.
dist:
	@mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		ext=$$( [ "$$os" = "windows" ] && echo .exe || echo "" ); \
		echo "  -> $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build -ldflags "$(LDFLAGS)" \
			-o dist/$(BINARY)-$$os-$$arch$$ext . ; \
	done
	@echo "Binaires dans dist/ :"
	@ls -1 dist/

clean:
	rm -rf bin dist
