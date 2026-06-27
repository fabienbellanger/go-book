package main

import (
	"fmt"
	"reflect"
	"strconv"
)

// Server : type de démonstration avec tags de struct et une méthode.
type Server struct {
	Host string `field:"host" default:"localhost"`
	Port int    `field:"port" default:"8080"`
	name string // non exporté : invisible à la sérialisation
}

// Addr renvoie l'adresse complète ; sert à l'appel dynamique.
func (s Server) Addr() string { return fmt.Sprintf("%s:%d", s.Host, s.Port) }

// FieldInfo décrit un champ exporté découvert par réflexion.
type FieldInfo struct {
	Name string
	Type string
	Tag  string
}

// InspectFields liste les champs EXPORTÉS d'une struct via l'itérateur Type.Fields()
// (Go 1.26), avec le tag `field` de chacun. C'est le socle d'un encodeur (JSON, CSV…).
func InspectFields(v any) []FieldInfo {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Struct {
		return nil
	}
	var out []FieldInfo
	for f := range t.Fields() { // itérateur 1.26 : iter.Seq[StructField]
		if !f.IsExported() {
			continue
		}
		out = append(out, FieldInfo{f.Name, f.Type.String(), f.Tag.Get("field")})
	}
	return out
}

// FillDefaults remplit les champs à zéro d'une struct à partir des tags `default`.
// Démontre l'ÉCRITURE par réflexion : il faut un POINTEUR (pour l'adressabilité) et
// vérifier CanSet. Value.Fields() (1.26) livre le couple (StructField, Value).
func FillDefaults(ptr any) error {
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("FillDefaults attend un *struct non nil, reçu %T", ptr)
	}
	for sf, fv := range v.Elem().Fields() { // (StructField, Value)
		def, ok := sf.Tag.Lookup("default")
		if !ok || !fv.CanSet() || !fv.IsZero() {
			continue // pas de défaut, non modifiable, ou déjà renseigné
		}
		switch fv.Kind() {
		case reflect.String:
			fv.SetString(def)
		case reflect.Int, reflect.Int64:
			n, err := strconv.ParseInt(def, 10, 64)
			if err != nil {
				return fmt.Errorf("tag default %q invalide pour %s : %w", def, sf.Name, err)
			}
			fv.SetInt(n)
		}
	}
	return nil
}

// CallMethod appelle dynamiquement une méthode par son nom et renvoie ses résultats.
func CallMethod(recv any, name string, args ...any) ([]any, error) {
	m := reflect.ValueOf(recv).MethodByName(name)
	if !m.IsValid() {
		return nil, fmt.Errorf("méthode %q absente sur %T", name, recv)
	}
	in := make([]reflect.Value, len(args))
	for i, a := range args {
		in[i] = reflect.ValueOf(a)
	}
	out := m.Call(in)
	res := make([]any, len(out))
	for i, o := range out {
		res[i] = o.Interface()
	}
	return res, nil
}
