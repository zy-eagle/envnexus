package repository

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewDB initializes a new GORM DB connection
func NewDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to MySQL database")
	return db, nil
}
