package main

import (
	"log"

	"github.com/YannHulot/petstore/api/models"
	"github.com/YannHulot/petstore/api/server"
	"github.com/jinzhu/gorm"
)

func main() {
	var db *gorm.DB

	// Load the .env file containing the variables
	err := models.LoadEnvFile()
	if err != nil {
		log.Fatalf("error while loading the env file: %s", err.Error())
	}

	// create a new config
	config, err := models.NewConfig()
	if err != nil {
		log.Fatalf("error while getting the config: %s", err.Error())
	}

	// validate the config
	err = config.Validate()
	if err != nil {
		log.Fatalf("error while validating the config: %s", err.Error())
	}

	// create the Database if possible
	err = models.CreatePgDb(config)
	if err != nil {
		log.Printf("error while creating the database: %s", err.Error())
		log.Printf("db may already exist")

		// database may already exist , so we can try to connect to it
		db, err = models.OpenAndTestDBConnection(config)
		if err != nil {
			// connection has failed for some reason
			// needs investigation so we shut down the application
			log.Fatalf("error while creating a connection with the db: %s", err.Error())
		}
	}

	if db == nil {
		// create the tables, run the migrations and open a connection to the DB
		db, err = models.OpenAndTestDBConnection(config)
		if err != nil {
			log.Fatalf("error while creating a connection with the db: %s", err.Error())
		}
	}

	defer db.Close()

	// create the router and the routes
	router := server.CreateRouter(db)

	// start the server
	log.Fatal(router.Run(":8080"))
}
