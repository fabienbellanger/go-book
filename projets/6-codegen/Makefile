# Makefile du Projet 6 — enumgen (générateur de méthodes String())
#
# Cibles principales :
#   make build      compile le binaire local dans bin/enumgen
#   make install    installe enumgen dans $GOBIN (requis par « go generate »)
#   make generate   régénère les fichiers *_enum.go (go generate ./...)
#   make test       go test -race ./...
#   make vet        go vet ./...
#   make dist       cross-compile pour linux/darwin/windows (amd64 + arm64)
#   make clean      supprime bin/ et dist/

BINARY  := enumgen
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X example.com/enumgen/internal/cli.version=$(VERSION)

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build install generate test vet fmt dist clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

# « go generate » lance enumgen depuis le PATH : il faut donc l'installer d'abord.
install:
	go install -ldflags "$(LDFLAGS)" .

generate: install
	go generate ./...

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

# Cross-compilation pure Go (sans cgo) pour des binaires statiques.
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
