// Command ch42-encoding illustre la sérialisation (encoding/json, gob, csv)
// et les expressions régulières (regexp) de la bibliothèque standard.
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Event modélise un évènement sérialisable en JSON.
//
// Les règles de mapping passent par les *tags* de struct :
//   - "name"      : nom de clé JSON (sinon, le nom du champ exporté est repris).
//   - omitempty   : omet le champ s'il est "vide" (0, "", nil, slice/map vide).
//   - omitzero    : omet le champ s'il vaut sa valeur zéro (🆕 Go 1.24) — gère
//     proprement les types dont le zéro n'est pas "empty", comme time.Time.
//   - ",string"   : encode un nombre/booléen comme une chaîne JSON.
//   - "-"         : champ jamais sérialisé.
//
// ⚠️ Seuls les champs EXPORTÉS (majuscule) sont (dé)sérialisés. Un champ non
// exporté est ignoré silencieusement.
type Event struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Tags      []string  `json:"tags,omitempty"`      // omis si nil/vide
	CreatedAt time.Time `json:"created_at,omitzero"` // omis si zéro (1.24)
	Score     int64     `json:"score,string"`        // encodé "42" (chaîne)
	internal  string    // non exporté : jamais sérialisé
	Secret    string    `json:"-"` // exclu explicitement
}

// Temperature montre un type avec encodage JSON personnalisé : il s'encode en
// nombre suivi de l'unité, ex. `"21.5°C"`, au lieu de la représentation par défaut.
type Temperature float64

// MarshalJSON implémente json.Marshaler.
func (t Temperature) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "%q", fmt.Sprintf("%.1f°C", float64(t))), nil
}

// UnmarshalJSON implémente json.Unmarshaler (parse "21.5°C" -> 21.5).
func (t *Temperature) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	s = strings.TrimSuffix(s, "°C")
	var v float64
	if _, err := fmt.Sscanf(s, "%g", &v); err != nil {
		return fmt.Errorf("température invalide %q: %w", s, err)
	}
	*t = Temperature(v)
	return nil
}

// marshalEvent sérialise un Event avec indentation lisible.
func marshalEvent(e Event) (string, error) {
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// strictDecode décode du JSON en refusant les champs inconnus : utile pour
// valider une entrée d'API (un champ en trop = erreur, pas un silence).
func strictDecode(src string) (Event, error) {
	dec := json.NewDecoder(strings.NewReader(src))
	dec.DisallowUnknownFields()
	var e Event
	if err := dec.Decode(&e); err != nil {
		return Event{}, err
	}
	return e, nil
}

// gobRoundTrip encode puis décode une valeur via encoding/gob (format binaire
// auto-décrit Go <-> Go, idéal entre deux programmes Go).
func gobRoundTrip(in Event) (Event, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(in); err != nil {
		return Event{}, err
	}
	var out Event
	if err := gob.NewDecoder(&buf).Decode(&out); err != nil {
		return Event{}, err
	}
	return out, nil
}

// csvSum lit un CSV "name,score" et renvoie la somme des scores. FieldsPerRecord
// est implicitement fixé par la première ligne : un nombre de colonnes variable
// déclenche une erreur.
func csvSum(src string) (int, error) {
	r := csv.NewReader(strings.NewReader(src))
	rows, err := r.ReadAll()
	if err != nil {
		return 0, err
	}
	sum := 0
	for _, row := range rows {
		var n int
		if _, err := fmt.Sscanf(row[1], "%d", &n); err != nil {
			return 0, fmt.Errorf("score invalide %q: %w", row[1], err)
		}
		sum += n
	}
	return sum, nil
}

// slugPattern est compilé UNE SEULE FOIS au niveau package. Recompiler une
// regexp dans une boucle chaude est un anti-patron coûteux (⚡).
var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

// slugify normalise un titre en slug ASCII (regexp + strings).
func slugify(title string) string {
	s := strings.ToLower(title)
	s = slugPattern.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// kvPattern utilise des groupes NOMMÉS (?P<name>...) pour extraire des paires.
var kvPattern = regexp.MustCompile(`(?P<key>\w+)=(?P<value>\w+)`)

// parseKV extrait les paires clé=valeur d'une chaîne grâce aux groupes nommés.
func parseKV(s string) map[string]string {
	out := map[string]string{}
	names := kvPattern.SubexpNames() // [ "", "key", "value" ]
	for _, m := range kvPattern.FindAllStringSubmatch(s, -1) {
		fields := map[string]string{}
		for i, val := range m {
			if names[i] != "" {
				fields[names[i]] = val
			}
		}
		out[fields["key"]] = fields["value"]
	}
	return out
}

func main() {
	e := Event{ID: 1, Name: "deploy", Tags: []string{"prod"}, Score: 42, Secret: "x"}
	js, _ := marshalEvent(e)
	fmt.Println("JSON:\n" + js)

	back, _ := gobRoundTrip(e)
	fmt.Printf("gob round-trip: %+v\n", back.Name)

	sum, _ := csvSum("alice,10\nbob,32\n")
	fmt.Println("csv sum:", sum)

	fmt.Println("slug:", slugify("Bonjour, le Monde !"))
	fmt.Printf("kv: %v\n", parseKV("env=prod region=eu"))

	var t Temperature = 21.5
	tj, _ := json.Marshal(t)
	fmt.Println("temp JSON:", string(tj))
}
