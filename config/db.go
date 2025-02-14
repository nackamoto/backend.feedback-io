package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	pwd, _ := os.Getwd()
	err := godotenv.Load(filepath.Join(pwd, ".env"))
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	db_conn, db_err := ConnectDB()
	if db_err != nil {
		log.Fatalf("Error connecting to database: %v", err)

	}

	// AutoMigrateDB(db_conn)
	// seeder.Seed(db_conn)

	DB = db_conn

	// defer func() {
	// 	db, err := DB.DB()
	// 	if err != nil {
	// 		log.Fatalf("Failed to get database instance: %v", err)

	// 	}
	// 	db.Close()
	// }()

}
