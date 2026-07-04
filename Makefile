# Makefile du livre — pilote le générateur de site (module tools/site).
#
# Lancé depuis la racine du dépôt, il entre dans tools/site pour exécuter
# le générateur (chemins relatifs conservés).
#
# Cibles principales :
#   make build   génère le site HTML dans public/
#   make serve   génère puis sert le site en local (http://localhost:8180)
#   make test    go test -race ./... du générateur
#   make check   fmt + vet + test (porte de qualité)
#   make chroma  régénère tools/site/assets/css/chroma.css (coloration)
#   make dist    compile le générateur (binaire statique)
#   make clean   supprime public/ et tools/site/bin/

# SITE : dossier du module générateur
# SRC  : racine du livre (relative à SITE, où sont chapitres/, annexes/)
# OUT  : dossier de sortie du site (relatif à SITE)
SITE    := tools/site
BINARY  := gobook-site
SRC     := ../..
OUT     := $(SRC)/public
ADDR    ?= :8180
LDFLAGS := -s -w

.PHONY: build serve test vet fmt update-deps check chroma dist clean

build:
	cd $(SITE) && go run . -clean -src $(SRC) -out $(OUT)

serve:
	cd $(SITE) && go run . -src $(SRC) -out $(OUT) -serve -addr $(ADDR)

test:
	cd $(SITE) && go test -race ./...

vet:
	cd $(SITE) && go vet ./...

fmt:
	cd $(SITE) && gofmt -l .

update-deps:
	cd $(SITE) && go get -u ./... && go mod tidy && go mod verify

check: fmt vet test

# Régénère la feuille de coloration syntaxique depuis les thèmes chroma.
chroma:
	cd $(SITE) && go run internal/render/gen_chroma/main.go > assets/css/chroma.css

# Binaire statique du générateur (pratique en CI : build une fois, exécute partout).
dist:
	@mkdir -p $(SITE)/bin
	cd $(SITE) && CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

clean:
	rm -rf $(SITE)/bin public
