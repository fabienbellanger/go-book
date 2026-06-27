package main

import (
	"fmt"
	"reflect"
)

func main() {
	// Introspection des champs (Type.Fields, 1.26).
	fmt.Println("champs de Server :")
	for _, f := range InspectFields(Server{}) {
		fmt.Printf("  %-5s %-7s tag=%q\n", f.Name, f.Type, f.Tag)
	}

	// Écriture par réflexion : remplir les défauts.
	var s Server
	if err := FillDefaults(&s); err != nil {
		fmt.Println("erreur:", err)
	}
	fmt.Printf("après FillDefaults : %+v\n", s)

	// Un champ déjà renseigné n'est pas écrasé.
	s2 := Server{Host: "example.com"}
	_ = FillDefaults(&s2)
	fmt.Printf("Host préexistant préservé : %s:%d\n", s2.Host, s2.Port)

	// Appel dynamique.
	res, _ := CallMethod(s, "Addr")
	fmt.Printf("appel dynamique Addr() -> %v\n", res[0])

	// Signature via Method.Type.Ins()/Outs() (1.26).
	mt := reflect.TypeOf(Server{}).Method(0).Type
	fmt.Print("signature Addr : in=[")
	for in := range mt.Ins() {
		fmt.Printf("%s ", in)
	}
	fmt.Print("] out=[")
	for out := range mt.Outs() {
		fmt.Printf("%s ", out)
	}
	fmt.Println("]")
}
