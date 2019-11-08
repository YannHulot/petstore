package models

import (
	"bytes"
	"log"
	"os/exec"

	"github.com/jinzhu/gorm"
)

// CreatePgDb will create a new db named pet_store
func CreatePgDb(c Config) error {
	cmd := exec.Command("createdb", "-p", "5432", "-h", c.DbHost, "-U", c.DbUser, "-e", c.DbName)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		log.Printf("Error: %v", err)
		return err
	}
	log.Printf("Output: %q\n", out.String())
	return nil
}

// OpenAndTestDBConnection will open a new connection and test it
func OpenAndTestDBConnection(c Config) (*gorm.DB, error) {
	dbURL := c.getDBConnectionURL()
	DB, err := gorm.Open(c.DbDriver, dbURL)
	if err != nil {
		return nil, err
	}

	DB.CreateTable()
	DB.Debug().AutoMigrate(&Pet{}, &Category{}, &Tag{})

	return DB, nil
}
