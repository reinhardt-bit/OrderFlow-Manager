// shared/db/db.go
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

	// Create products table
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS products (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            price REAL NOT NULL,
            active BOOLEAN DEFAULT true
        )
    `)
	if err != nil {
		return nil, fmt.Errorf("error creating products table: %v", err)
	}

	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS representatives (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        active BOOLEAN DEFAULT true
    )
`)
	if err != nil {
		return nil, fmt.Errorf("error creating representatives table: %v", err)
	}

	// Create orders table
	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS orders (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        created_at DATETIME,
        client_name TEXT,
        contact TEXT,
        product_id INTEGER,
        quantity INTEGER,
        price REAL,
        needs_delivery BOOLEAN,
        delivery_address TEXT,
        comment TEXT,
        completed BOOLEAN,
        representative_id INTEGER,  -- Add the column first
        FOREIGN KEY(product_id) REFERENCES products(id),
        FOREIGN KEY(representative_id) REFERENCES representatives(id)
    )
`)
	if err != nil {
		return nil, fmt.Errorf("error creating orders table: %v", err)
	}

	return db, nil
}
