package models

import "testing"

func TestPetSanitizing(t *testing.T) {
	newPet := Pet{}
	newPet.Name = "\"\""

	newPet.Sanitise()

	if newPet.Name != "&#34;&#34;" {
		t.Log(newPet.Name)
		t.Errorf("a")
	}
}
