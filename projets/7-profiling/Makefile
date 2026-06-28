# Makefile du Projet 7 — wordstats (profiling de bout en bout)
#
# Cibles principales :
#   make build      compile bin/wordstats (PGO auto si default.pgo présent)
#   make run        lance le service (make run ARGS="-addr :8080 -slow 50ms")
#   make test       go test -race ./...
#   make bench      compare v1 vs v2 (-benchmem) — le « avant/après »
#   make benchstat  bench formaté pour l'outil benchstat
#   make cpu        profil CPU de la v1, ouvre le top pprof
#   make mem        profil mémoire de la v1, ouvre le top pprof
#   make pgo        (re)génère default.pgo puis reconstruit avec PGO
#   make trace      lance le service avec capture de trace sur requête lente
#   make vet / fmt  go vet ./... / gofmt -l .
#   make clean      supprime bin/, dist/ et les profils

BINARY  := wordstats
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
PKG     := ./internal/analyze

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build run test bench benchstat cpu mem pgo trace vet fmt dist clean

# go build applique automatiquement default.pgo s'il existe (-pgo=auto par défaut).
build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

run:
	go run . $(ARGS)

test:
	go test -race ./...

# Le « avant/après » : les deux implémentations, mémoire comprise.
bench:
	go test -run=^$$ -bench=TopWords -benchmem -count=10 $(PKG)

# Sortie brute destinée à « benchstat » (installer : go install golang.org/x/perf/cmd/benchstat@latest).
benchstat:
	go test -run=^$$ -bench=TopWords -benchmem -count=10 $(PKG) | tee bench.txt
	@echo "Comparer : benchstat -col /impl bench.txt  (ou old.txt new.txt)"

# Profil CPU de la version naïve : on voit la regexp dominer.
cpu:
	go test -run=^$$ -bench=BenchmarkTopWordsRegexp -cpuprofile=cpu.pprof $(PKG)
	go tool pprof -top -nodecount=15 cpu.pprof

# Profil mémoire de la version naïve : on voit les allocations exploser.
mem:
	go test -run=^$$ -bench=BenchmarkTopWordsRegexp -memprofile=mem.pprof $(PKG)
	go tool pprof -top -sample_index=alloc_space -nodecount=15 mem.pprof

# PGO : on collecte un profil du chemin chaud (v2) et on le fige en default.pgo,
# puis on reconstruit (go build le détecte tout seul).
pgo:
	go test -run=^$$ -bench=BenchmarkTopWordsScan -cpuprofile=cpu-scan.pprof $(PKG)
	cp cpu-scan.pprof default.pgo
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .
	@go version -m bin/$(BINARY) | grep -i pgo || true

# Démonstration du FlightRecorder : toute requête au-dessus de -slow capture une trace.
trace: build
	@echo "Service sur :8080, traces dans ./ ; Ctrl-C pour arrêter."
	./bin/$(BINARY) -addr :8080 -slow 20ms -tracedir .

vet:
	go vet ./...

fmt:
	gofmt -l .

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

clean:
	rm -rf bin dist
	rm -f cpu.pprof mem.pprof cpu-scan.pprof bench.txt *.trace
