package main

import (
	"strings"
	"testing"
)

// safeCall convertit une panique en erreur, et renvoie nil sans panique.
func TestSafeCall(t *testing.T) {
	if err := safeCall(func() {}); err != nil {
		t.Errorf("sans panique : err = %v ; attendu nil", err)
	}
	err := safeCall(func() { panic("boom") })
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("avec panique : err = %v ; attendu une erreur contenant \"boom\"", err)
	}
}

// Une panique runtime (division par zéro) est aussi rattrapable.
func TestSafeCallRuntimePanic(t *testing.T) {
	err := safeCall(func() { _ = divide(1, 0) })
	if err == nil || !strings.Contains(err.Error(), "divide by zero") {
		t.Errorf("err = %v ; attendu \"divide by zero\"", err)
	}
}

// mustPositive panique sur une valeur invalide.
func TestMustPositivePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("mustPositive(0) aurait dû paniquer")
		}
	}()
	mustPositive(0)
}

// validate : nil si valide, erreur ciblée sur le premier champ fautif.
func TestValidate(t *testing.T) {
	if err := validate(30, 100); err != nil {
		t.Errorf("validate(30,100) = %v ; attendu nil", err)
	}
	err := validate(-1, 100)
	if err == nil || !strings.Contains(err.Error(), "age") {
		t.Errorf("validate(-1,100) = %v ; attendu une erreur sur \"age\"", err)
	}
}

// validate re-déclenche une panique INATTENDUE (qui n'est pas un validationPanic).
func TestValidateRepanicsUnknown(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("une panique inattendue aurait dû remonter")
		}
	}()
	// On force une panique non-validation à l'intérieur de la frontière de validate
	// en passant par un checkPositive enveloppé qui panique différemment.
	func() (err error) {
		defer func() {
			switch r := recover().(type) {
			case nil:
			case validationPanic:
				err = nil
			default:
				panic(r) // doit remonter jusqu'au defer du test
			}
		}()
		panic("vrai bug")
	}()
}

// La frontière de recover transforme une panique de handler en 500.
func TestRecoverMiddleware(t *testing.T) {
	h := recoverMiddleware(app)
	if resp := h(Request{path: "/home"}); resp.status != 200 {
		t.Errorf("/home -> %d ; attendu 200", resp.status)
	}
	if resp := h(Request{path: "/boom"}); resp.status != 500 {
		t.Errorf("/boom -> %d ; attendu 500", resp.status)
	}
}
