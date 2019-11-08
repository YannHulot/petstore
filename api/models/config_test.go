package models

import (
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestNewConfigValidation(t *testing.T) {
	os.Setenv("DB_PASSWORD", "test2")
	os.Setenv("DB_PORT", "test3")
	os.Setenv("DB_HOST", "test4")
	os.Setenv("DB_NAME", "test5")
	os.Setenv("DB_DRIVER", "test6")

	c, err := NewConfig()
	if err != nil {
		spew.Dump(err)
		t.Fatal("there should be no errors creating the config")
	}

	err = c.Validate()
	if err == nil {
		spew.Dump(err)
		t.Fatal("config should be invalid")
	}

	os.Setenv("DB_USER", "test1")
	c2, err := NewConfig()
	if err != nil {
		spew.Dump(err)
		t.Fatal("there should be no errors creating the config")
	}

	err = c2.Validate()
	if err != nil {
		t.Fatal("config should be valid")
	}
}
