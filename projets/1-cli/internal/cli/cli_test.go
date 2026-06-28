package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// readCloserFrom fabrique un open() de source à partir d'une chaîne en mémoire.
func readCloserFrom(content string) func() (io.ReadCloser, error) {
	return func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(content)), nil
	}
}

// run est un helper de test : il exécute Run avec une entrée stdin donnée et
// capture stdout, stderr et le code de retour.
func run(t *testing.T, stdin string, args ...string) (out, errOut string, code int) {
	t.Helper()
	var bo, be bytes.Buffer
	code = Run(args, strings.NewReader(stdin), &bo, &be)
	return bo.String(), be.String(), code
}

func TestRunDispatch(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string // sous-chaîne attendue sur stdout
		wantErr  string // sous-chaîne attendue sur stderr
	}{
		{"sans argument", nil, 2, "", "Usage"},
		{"version", []string{"version"}, 0, "txtkit dev", ""},
		{"help", []string{"help"}, 0, "Commandes", ""},
		{"inconnue", []string{"frobnicate"}, 2, "", "sous-commande inconnue"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, code := run(t, "", tt.args...)
			if code != tt.wantCode {
				t.Errorf("code = %d, voulu %d", code, tt.wantCode)
			}
			if tt.wantOut != "" && !strings.Contains(out, tt.wantOut) {
				t.Errorf("stdout = %q, doit contenir %q", out, tt.wantOut)
			}
			if tt.wantErr != "" && !strings.Contains(errOut, tt.wantErr) {
				t.Errorf("stderr = %q, doit contenir %q", errOut, tt.wantErr)
			}
		})
	}
}
