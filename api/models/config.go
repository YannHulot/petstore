package models

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config is the configuration for the database
type Config struct {
	DbUser     string
	DbPassword string
	DbPort     string
	DbHost     string
	DbName     string
	DbDriver   string
}

// Validate will validate the config and make sure that all the env variables needed to establish the connection
// with the DB are present in the environment
func (c *Config) Validate() error {
	if c.DbUser == "" {
		return fmt.Errorf("DbUser is empty")
	}

	if c.DbPassword == "" {
		return fmt.Errorf("DbPassword is empty")
	}

	if c.DbPort == "" {
		return fmt.Errorf("DbPort is empty")
	}

	if c.DbHost == "" {
		return fmt.Errorf("DbHost is empty")
	}

	if c.DbName == "" {
		return fmt.Errorf("DbName is empty")
	}

	if c.DbDriver == "" {
		return fmt.Errorf("DbDriver is empty")
	}
	return nil
}

func (c *Config) getDBConnectionURL() string {
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable password=%s",
		c.DbHost, c.DbPort, c.DbUser, c.DbName, c.DbPassword)
}

// NewConfig will return a new config
func NewConfig() (Config, error) {
	DbUser := os.Getenv("DB_USER")
	DbPassword := os.Getenv("DB_PASSWORD")
	DbPort := os.Getenv("DB_PORT")
	DbHost := os.Getenv("DB_HOST")
	DbName := os.Getenv("DB_NAME")
	DbDriver := os.Getenv("DB_DRIVER")

	return Config{
		DbUser,
		DbPassword,
		DbPort,
		DbHost,
		DbName,
		DbDriver,
	}, nil
}

// LoadEnvFile should load the .env file
func LoadEnvFile() error {
	err := godotenv.Load()
	if err != nil {
		return err
	}

	return nil
}
