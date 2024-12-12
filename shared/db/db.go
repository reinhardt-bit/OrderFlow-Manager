package db

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func InitDB() (*sql.DB, error) {
	err := godotenv.Load(".env")
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}

	primaryUrl := os.Getenv("TURSO_DATABASE_URL")
	if primaryUrl == "" {
		return nil, fmt.Errorf("TURSO_DATABASE_URL not set")
	}

	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	if authToken == "" {
		return nil, fmt.Errorf("TURSO_AUTH_TOKEN not set")
	}

	db, err := sql.Open("libsql", primaryUrl+"?authToken="+authToken)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}

	// Modified table creation with correct column names
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS orders (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            created_at DATETIME,
            client_name TEXT,
            contact TEXT,
            select_product TEXT,
            select_flavor TEXT,
            quantity INTEGER,
            price REAL,
            needs_delivery BOOLEAN,
            delivery_address TEXT,
            comment TEXT,
            completed BOOLEAN
        )
    `)
	if err != nil {
		return nil, fmt.Errorf("error creating tables: %v", err)
	}

	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS products (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        base_price REAL NOT NULL
    )
`)
	if err != nil {
		return nil, fmt.Errorf("error creating products table: %v", err)
	}

	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS flavors (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        product_id INTEGER,
        FOREIGN KEY(product_id) REFERENCES products(id)
    )
`)
	if err != nil {
		return nil, fmt.Errorf("error creating flavors table: %v", err)
	}

	return db, nil
}
