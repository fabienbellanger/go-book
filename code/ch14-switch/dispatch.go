package main

// levelFromString : switch de CHAÎNES. Le compilateur le compile en regroupement par
// longueur puis recherche binaire — souvent plus rapide qu'une map pour un petit
// ensemble fixe (ni hachage ni allocation).
func levelFromString(s string) int {
	switch s {
	case "trace":
		return 0
	case "debug":
		return 1
	case "info":
		return 2
	case "warn":
		return 3
	case "error":
		return 4
	case "fatal":
		return 5
	default:
		return -1
	}
}

// levelMap : la même table, version map (pour la comparaison de performance).
var levelMap = map[string]int{
	"trace": 0, "debug": 1, "info": 2, "warn": 3, "error": 4, "fatal": 5,
}

func levelFromMap(s string) int {
	if v, ok := levelMap[s]; ok {
		return v
	}
	return -1
}

// levelFromInt : switch d'entiers DENSE (0..7). Avec assez de cas, le compilateur le
// compile en JUMP TABLE — une borne puis un saut indirect. Inspecter avec :
// go build -gcflags='-S' ./ch14-switch/
// Les valeurs renvoyées sont distinctes des entrées pour éviter un repli en identité.
func levelFromInt(n int) int {
	switch n {
	case 0:
		return 100
	case 1:
		return 200
	case 2:
		return 300
	case 3:
		return 400
	case 4:
		return 500
	case 5:
		return 503
	case 6:
		return 600
	case 7:
		return 700
	default:
		return -1
	}
}
