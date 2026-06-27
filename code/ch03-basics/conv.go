package main

import "math"

// toInt8 convertit n en int8 SI la valeur tient dans l'intervalle [-128, 127],
// et signale par false tout débordement.
//
// Sans ce garde-fou, int8(n) tronquerait silencieusement les bits de poids fort
// (ex. int8(200) == -56) : c'est le piège du débordement à l'exécution.
func toInt8(n int) (int8, bool) {
	if n < math.MinInt8 || n > math.MaxInt8 {
		return 0, false
	}
	return int8(n), true
}
