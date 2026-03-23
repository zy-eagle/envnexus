package repository

import (
	"log/slog"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewDB initializes a new GORM DB connection
func NewDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	slog.Info("successfully connected to MySQL database")
	return db, nil
}
