// code/ch43-slog/slog_test.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// newTestLogger renvoie un logger JSON déterministe écrivant dans buf : niveau
// Debug, et l'horodatage retiré via ReplaceAttr pour des assertions stables.
func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && a.Key == slog.TimeKey {
				return slog.Attr{} // un Attr vide est supprimé de la sortie
			}
			return a
		},
	}
	return slog.New(contextHandler{slog.NewJSONHandler(buf, opts)})
}

// decodeLast décode la dernière ligne JSON écrite dans buf.
func decodeLast(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	last := lines[len(lines)-1]
	var m map[string]any
	if err := json.Unmarshal([]byte(last), &m); err != nil {
		t.Fatalf("JSON invalide %q : %v", last, err)
	}
	return m
}

func TestTypedAttrs(t *testing.T) {
	var buf bytes.Buffer
	newTestLogger(&buf).Info("hello", slog.String("addr", ":8080"))

	m := decodeLast(t, &buf)
	if m["level"] != "INFO" || m["msg"] != "hello" || m["addr"] != ":8080" {
		t.Fatalf("attributs inattendus : %v", m)
	}
	if _, ok := m["time"]; ok {
		t.Errorf("l'horodatage aurait dû être retiré : %v", m)
	}
}

func TestLogValuerRedactsSecret(t *testing.T) {
	var buf bytes.Buffer
	u := User{ID: 7, Name: "ada", Pass: "s3cr3t"}
	newTestLogger(&buf).Info("login", slog.Any("user", u))

	out := buf.String()
	if strings.Contains(out, "s3cr3t") {
		t.Fatalf("le secret a fuité dans les logs : %s", out)
	}
	m := decodeLast(t, &buf)
	user, ok := m["user"].(map[string]any)
	if !ok {
		t.Fatalf("user devrait être un groupe : %v", m)
	}
	if user["password"] != "[REDACTED]" {
		t.Errorf("password = %v, voulu [REDACTED]", user["password"])
	}
	if user["name"] != "ada" {
		t.Errorf("name = %v, voulu ada", user["name"])
	}
}

func TestContextHandlerInjectsRequestID(t *testing.T) {
	var buf bytes.Buffer
	ctx := withRequestID(context.Background(), "req-42")
	newTestLogger(&buf).InfoContext(ctx, "work")

	if got := decodeLast(t, &buf)["request_id"]; got != "req-42" {
		t.Errorf("request_id = %v, voulu req-42", got)
	}
}

func TestWithPrecomputesAttrs(t *testing.T) {
	var buf bytes.Buffer
	// .With() exerce WithAttrs : on vérifie en prime que notre wrapper survit
	// (le request_id du contexte doit toujours être injecté après .With).
	logger := newTestLogger(&buf).With(slog.String("component", "auth"))
	ctx := withRequestID(context.Background(), "req-99")
	logger.InfoContext(ctx, "a")
	logger.WarnContext(ctx, "b")

	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if !strings.Contains(line, `"component":"auth"`) {
			t.Errorf("attribut commun manquant : %s", line)
		}
		if !strings.Contains(line, `"request_id":"req-99"`) {
			t.Errorf("request_id perdu après .With (bug d'embedding ?) : %s", line)
		}
	}
}

// TestLevelVarHotSwap vérifie qu'on peut changer le seuil de log à chaud.
func TestLevelVarHotSwap(t *testing.T) {
	var buf bytes.Buffer
	level := new(slog.LevelVar) // Info par défaut
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: level}))

	logger.Debug("filtré")
	if buf.Len() != 0 {
		t.Fatalf("Debug n'aurait pas dû passer au niveau Info : %s", buf.String())
	}
	level.Set(slog.LevelDebug)
	logger.Debug("visible")
	if !strings.Contains(buf.String(), "visible") {
		t.Errorf("Debug aurait dû passer après abaissement du niveau : %s", buf.String())
	}
}

// TestBadKey illustre le piège du nombre impair d'arguments (clé sans valeur).
func TestBadKey(t *testing.T) {
	var buf bytes.Buffer
	// "oops" est une clé sans valeur : slog la signale par !BADKEY.
	// On passe les arguments via un slice pour démontrer le piège à l'exécution
	// (l'analyseur `go vet` détecte la forme littérale, pas celle-ci).
	args := []any{"oops"}
	slog.New(slog.NewJSONHandler(&buf, nil)).Info("msg", args...)
	if !strings.Contains(buf.String(), "BADKEY") {
		t.Errorf("un argument orphelin aurait dû produire !BADKEY : %s", buf.String())
	}
}
