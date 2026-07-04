package main

// ch02Greetings associe un code langue à sa salutation. Ajouter une langue se
// résume à ajouter une entrée — dont "de" (exercice 1).
var ch02Greetings = map[string]string{
	"fr": "Bonjour",
	"en": "Hello",
	"de": "Hallo",
}

// ch02Greet renvoie une salutation dans la langue demandée, avec repli sur "en"
// si la langue est inconnue. L'identifiant est exporté-en-anglais côté code ;
// c'est la MAJUSCULE initiale qui exporterait le symbole hors du paquet
// (exercice 2 : renommer en minuscule le rend invisible à un autre paquet).
func ch02Greet(lang, name string) string {
	msg, ok := ch02Greetings[lang]
	if !ok {
		msg = ch02Greetings["en"]
	}
	return msg + ", " + name + " !"
}
