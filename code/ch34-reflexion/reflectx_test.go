package main

import "testing"

func TestInspectFields(t *testing.T) {
	fields := InspectFields(Server{})
	if len(fields) != 2 { // Host, Port ; name est non exporté
		t.Fatalf("len = %d ; attendu 2 (champs exportés), got %+v", len(fields), fields)
	}
	if fields[0].Name != "Host" || fields[0].Tag != "host" {
		t.Errorf("champ 0 = %+v ; attendu Host/host", fields[0])
	}
	if fields[1].Name != "Port" || fields[1].Type != "int" {
		t.Errorf("champ 1 = %+v ; attendu Port/int", fields[1])
	}
}

func TestInspectFieldsNonStruct(t *testing.T) {
	if got := InspectFields(42); got != nil {
		t.Errorf("InspectFields(int) = %v ; attendu nil", got)
	}
}

func TestFillDefaults(t *testing.T) {
	var s Server
	if err := FillDefaults(&s); err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if s.Host != "localhost" || s.Port != 8080 {
		t.Errorf("après FillDefaults : %+v ; attendu localhost/8080", s)
	}
}

func TestFillDefaultsPreservesNonZero(t *testing.T) {
	s := Server{Host: "example.com"} // Port reste 0 -> sera rempli, Host non
	if err := FillDefaults(&s); err != nil {
		t.Fatal(err)
	}
	if s.Host != "example.com" {
		t.Errorf("Host = %q ; devait être préservé", s.Host)
	}
	if s.Port != 8080 {
		t.Errorf("Port = %d ; attendu 8080 (rempli car zéro)", s.Port)
	}
}

func TestFillDefaultsRejectsNonPointer(t *testing.T) {
	if err := FillDefaults(Server{}); err == nil {
		t.Error("FillDefaults(valeur) devrait échouer : besoin d'un pointeur")
	}
	if err := FillDefaults((*Server)(nil)); err == nil {
		t.Error("FillDefaults(pointeur nil) devrait échouer")
	}
}

func TestCallMethod(t *testing.T) {
	res, err := CallMethod(Server{Host: "h", Port: 9}, "Addr")
	if err != nil {
		t.Fatal(err)
	}
	if got := res[0].(string); got != "h:9" {
		t.Errorf("Addr() = %q ; attendu \"h:9\"", got)
	}
	if _, err := CallMethod(Server{}, "Inexistante"); err == nil {
		t.Error("appel d'une méthode absente devrait échouer")
	}
}
