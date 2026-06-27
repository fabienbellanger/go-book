package main

// grade note un score via un switch SANS condition (tagless) : chaque cas est une
// condition booléenne, évaluée de haut en bas. Idéal pour des intervalles.
func grade(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	default:
		return "F"
	}
}

// dayKind classe un jour via un switch d'EXPRESSION avec plusieurs valeurs par cas
// (séparées par des virgules = « OU »).
func dayKind(day string) string {
	switch day {
	case "samedi", "dimanche":
		return "week-end"
	case "vendredi":
		return "presque le week-end"
	default:
		return "semaine"
	}
}

// nameStatus illustre le switch avec instruction d'init : n n'existe que dans le switch.
func nameStatus(name string) string {
	switch n := len(name); {
	case n == 0:
		return "vide"
	case n > 64:
		return "trop long"
	default:
		return "ok"
	}
}

// capabilities renvoie les droits d'un rôle. fallthrough fait hériter chaque rôle des
// droits du rôle inférieur — il saute au corps du cas suivant SANS tester sa condition.
func capabilities(role string) []string {
	var caps []string
	switch role {
	case "admin":
		caps = append(caps, "delete")
		fallthrough
	case "editor":
		caps = append(caps, "write")
		fallthrough
	case "viewer":
		caps = append(caps, "read")
	}
	return caps
}
