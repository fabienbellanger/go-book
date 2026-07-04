// Command ch51-database-sql montre l'API database/sql de bout en bout :
// ouverture et réglage d'un pool, requêtes paramétrées, Scan, gestion de
// sql.ErrNoRows, transactions Commit/Rollback et annulation par contexte.
//
// Le driver "memdb" (voir memdb.go) est factice et en mémoire : il rend ces
// exemples exécutables sans base réelle, tout en montrant que database/sql est
// une API abstraite — le code ci-dessous ne dépend d'aucun moteur particulier.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

// User modélise une ligne de la table users.
type User struct {
	ID    int64
	Name  string
	Email string
}

// openDB ouvre le pool et le configure. sql.Open NE SE CONNECTE PAS : il
// prépare un pool paresseux (les connexions naissent à la première requête).
// PingContext force une prise de connexion pour valider le DSN au démarrage.
func openDB(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("memdb", dsn)
	if err != nil {
		return nil, err
	}
	// Réglages du pool : déterminants en production (voir le chapitre).
	db.SetMaxOpenConns(10)                  // plafond de connexions simultanées
	db.SetMaxIdleConns(5)                   // connexions gardées au chaud
	db.SetConnMaxLifetime(30 * time.Minute) // recyclage périodique
	db.SetConnMaxIdleTime(5 * time.Minute)  // fermeture des connexions oisives

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// insertUser insère un utilisateur et renvoie son id via LastInsertId.
func insertUser(ctx context.Context, db *sql.DB, name, email string) (int64, error) {
	res, err := db.ExecContext(ctx,
		"insert into users(name, email) values (?, ?)", name, email)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// getUser lit un utilisateur par id. L'ABSENCE de ligne est signalée par
// sql.ErrNoRows — un cas NORMAL, à distinguer d'une vraie erreur d'exécution.
func getUser(ctx context.Context, db *sql.DB, id int64) (User, error) {
	var u User
	err := db.QueryRowContext(ctx,
		"select id, name, email from users where id = ?", id).
		Scan(&u.ID, &u.Name, &u.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, fmt.Errorf("utilisateur %d introuvable : %w", id, err)
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}

// listUsers lit toutes les lignes. Le trio « defer rows.Close() + boucle
// rows.Next/Scan + rows.Err() après la boucle » est OBLIGATOIRE : sans
// rows.Err(), une erreur survenue en cours d'itération passerait inaperçue.
func listUsers(ctx context.Context, db *sql.DB) ([]User, error) {
	rows, err := db.QueryContext(ctx, "select id, name, email from users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err() // erreur SURVENUE PENDANT l'itération
}

// renameInTx renomme deux utilisateurs dans une SEULE transaction : soit les
// deux renommages sont validés, soit aucun. Le patron « defer tx.Rollback() »
// garantit l'annulation si un chemin d'erreur oublie de committer ; après un
// Commit réussi, ce Rollback est un no-op (il renvoie sql.ErrTxDone, ignoré).
func renameInTx(ctx context.Context, db *sql.DB, id1, id2 int64, name1, name2 string) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelDefault})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		"update users set name = ? where id = ?", name1, id1); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"update users set name = ? where id = ?", name2, id2); err != nil {
		return err
	}
	return tx.Commit()
}

func main() {
	ctx := context.Background()
	db, err := openDB(ctx, "demo")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	alice, err := insertUser(ctx, db, "Alice", "alice@example.com")
	if err != nil {
		log.Fatal(err)
	}
	bob, err := insertUser(ctx, db, "Bob", "bob@example.com")
	if err != nil {
		log.Fatal(err)
	}

	u, err := getUser(ctx, db, alice)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("lu : %+v\n", u)

	if err := renameInTx(ctx, db, alice, bob, "Alice A.", "Bob B."); err != nil {
		log.Fatal(err)
	}

	users, err := listUsers(ctx, db)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d utilisateur(s) après la transaction :\n", len(users))
	for _, u := range users {
		fmt.Printf("  %d %s <%s>\n", u.ID, u.Name, u.Email)
	}

	// Un id absent → sql.ErrNoRows, encapsulé par getUser.
	if _, err := getUser(ctx, db, 999); err != nil {
		fmt.Println("attendu :", err)
	}
}
