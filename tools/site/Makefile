# Makefile du générateur de site du livre
#
# Cibles principales :
#   make build   génère le site HTML dans ../../public
#   make serve   génère puis sert le site en local (http://localhost:8080)
#   make test    go test -race ./...
#   make check   fmt + vet + test (porte de qualité)
#   make chroma  régénère assets/css/chroma.css (coloration)
#   make dist    compile le générateur (binaire statique)
#   make clean   supprime public/ et bin/

BINARY  := gobook-site
SRC     := ../..                       # racine du livre (où sont chapitres/, annexes/)
OUT     := $(SRC)/public               # dossier de sortie du site
ADDR    ?= :8080
LDFLAGS := -s -w

.PHONY: build serve test vet fmt update-deps check chroma dist clean

build:
	go run . -clean -src $(SRC) -out $(OUT)

serve:
	go run . -src $(SRC) -out $(OUT) -serve -addr $(ADDR)

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

update-deps:
	go get -u ./...
	go mod tidy
	go mod verify

check: fmt vet test

# Régénère la feuille de coloration syntaxique depuis les thèmes chroma.
chroma:
	go run internal/render/gen_chroma/main.go > assets/css/chroma.css

# Binaire statique du générateur (pratique en CI : build une fois, exécute partout).
dist:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

clean:
	rm -rf bin $(OUT)
