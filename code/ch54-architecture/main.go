// Command ch54-architecture illustre le CÂBLAGE (composition root) d'une petite
// application : main construit les dépendances concrètes et les injecte, de la
// périphérie (store) vers le cœur (service). C'est le SEUL endroit qui connaît
// à la fois les types concrets et la façon de les assembler.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"example.com/gobook/ch54-architecture/service"
	"example.com/gobook/ch54-architecture/store"
)

func main() {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Câblage explicite : le store concret est injecté dans le service, qui ne
	// le voit qu'à travers son interface NoteStore. Changer de store (SQL,
	// fichier) ne modifierait QUE cette ligne.
	st := store.NewMem()
	svc := service.New(st, log)

	n, err := svc.Create(ctx, "  Première note  ", "Bonjour Go")
	if err != nil {
		fmt.Println("erreur:", err)
		return
	}

	got, err := svc.Get(ctx, n.ID)
	if err != nil {
		fmt.Println("erreur:", err)
		return
	}
	fmt.Printf("relue: %s / %q\n", got.ID, got.Title)
}
