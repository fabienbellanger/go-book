// Code généré par enumgen ; NE PAS MODIFIER.

package example

import "strconv"

var _Color_names = map[Color]string{
	ColorRed:   "Red",
	ColorGreen: "Green",
	ColorBlue:  "Blue",
}

// String implémente fmt.Stringer pour Color.
func (v Color) String() string {
	if s, ok := _Color_names[v]; ok {
		return s
	}
	return "Color(" + strconv.FormatInt(int64(v), 10) + ")"
}

var _Priority_names = map[Priority]string{
	PriorityLow:      "Low",
	PriorityMedium:   "Medium",
	PriorityHigh:     "High",
	PriorityCritical: "Critical",
}

// String implémente fmt.Stringer pour Priority.
func (v Priority) String() string {
	if s, ok := _Priority_names[v]; ok {
		return s
	}
	return "Priority(" + strconv.FormatInt(int64(v), 10) + ")"
}
