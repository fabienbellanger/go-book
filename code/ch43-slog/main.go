// code/ch43-slog/main.go
// Démonstration de la journalisation structurée avec log/slog :
// handlers texte/JSON, attributs typés, contexte commun via With/WithGroup,
// rédaction d'un secret via LogValuer, et niveau ajustable à chaud (LevelVar).
package main

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// Password est un type sensible : il NE doit JAMAIS apparaître en clair dans les
// logs. En implémentant slog.LogValuer, il décide lui-même de ce qui est journalisé.
type Password string

// LogValue masque la valeur : slog appelle cette méthode au moment de logguer.
// C'est aussi un point d'extension « paresseux » (lazy) : le calcul n'a lieu que
// si l'enregistrement est effectivement émis.
func (Password) LogValue() slog.Value { return slog.StringValue("[REDACTED]") }

// User regroupe des champs de domaine. On l'expose comme un groupe d'attributs.
type User struct {
	ID   int
	Name string
	Pass Password
}

// LogValue transforme un User en groupe d'attributs structurés. Le mot de passe
// passe par son propre LogValuer : il reste masqué même imbriqué.
func (u User) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("id", u.ID),
		slog.String("name", u.Name),
		slog.Any("password", u.Pass),
	)
}

// ctxKey est un type non exporté pour les clés de contexte (cf. Ch. 22).
type ctxKey int

const requestIDKey ctxKey = iota

// withRequestID attache un identifiant de requête au contexte.
func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// contextHandler enrichit chaque enregistrement avec les champs de portée requête
// trouvés dans le contexte (ici, request_id). Il délègue tout le reste au handler
// englobé : un handler personnalisé léger se résume souvent à cela.
type contextHandler struct{ slog.Handler }

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

// ⚠️ WithAttrs et WithGroup DOIVENT ré-emballer : sans cela, les méthodes promues
// par l'embedding renverraient le handler interne nu, et notre Handle (donc le
// request_id) serait perdu après chaque appel à Logger.With.
func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}

// newLogger construit un logger JSON, niveau réglable à chaud, qui injecte le
// request_id du contexte.
func newLogger(opts *slog.HandlerOptions) (*slog.Logger, *slog.LevelVar) {
	level := new(slog.LevelVar) // zéro = LevelInfo
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	opts.Level = level
	base := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(contextHandler{base}), level
}

func main() {
	logger, level := newLogger(nil)

	// Attributs typés : pas de boxing, clés explicites.
	logger.Info("service démarré",
		slog.String("addr", ":8080"),
		slog.Duration("timeout", 5*time.Second),
	)

	// Contexte commun pré-calculé via With : réutilisé pour chaque message.
	reqLog := logger.With(slog.String("component", "auth"))

	ctx := withRequestID(context.Background(), "req-42")
	u := User{ID: 7, Name: "ada", Pass: "s3cr3t"}

	// InfoContext : le handler lit le request_id dans ctx ; le mot de passe est masqué.
	reqLog.InfoContext(ctx, "connexion", slog.Any("user", u))

	// Debug est filtré tant que le niveau est Info...
	reqLog.DebugContext(ctx, "détail interne", slog.Int("attempt", 1))

	// ...on abaisse le seuil à chaud : le prochain Debug passe.
	level.Set(slog.LevelDebug)
	reqLog.DebugContext(ctx, "détail interne", slog.Int("attempt", 2))
}
