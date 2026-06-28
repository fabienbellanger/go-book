# Makefile du Projet 4 — gends (bibliothèque générique)
#
# Une bibliothèque ne produit pas de binaire : les cibles tournent autour des
# tests, des exemples, des benchmarks et du fuzzing.
#
#   make test       go test -race ./...    (inclut les Example testables)
#   make cover      couverture + rapport HTML
#   make bench      lance tous les benchmarks
#   make fuzz       fuzze le cache LRU (FUZZTIME réglable)
#   make vet / fmt  go vet / vérification gofmt
#   make ci         enchaîne fmt + vet + test (cf. CI GitHub Actions)

FUZZTIME ?= 30s

.PHONY: test cover bench fuzz vet fmt ci

test:
	go test -race ./...

cover:
	go test -race -coverprofile=cover.out ./...
	go tool cover -func=cover.out | tail -1
	go tool cover -html=cover.out -o cover.html
	@echo "Rapport : cover.html"

bench:
	go test -run=^$$ -bench=. -benchmem ./...

fuzz:
	go test -run=Fuzz -fuzz=FuzzCache -fuzztime=$(FUZZTIME) ./lru

vet:
	go vet ./...

fmt:
	@test -z "$$(gofmt -l .)" || { echo "Fichiers non formatés :"; gofmt -l .; exit 1; }

ci: fmt vet test
