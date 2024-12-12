// shared/db/db.go
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func InitDB() (*sql.DB, error) {
	// First, try to update environment variables from the config file
	if err := UpdateEnvForDbConfig(); err != nil {
		log.Printf("Warning: Could not update config from file: %v", err)
	}

	// Validate database configuration
	if err := ValidateDbConfig(); err != nil {
		return nil, fmt.Errorf("database configuration invalid: %v", err)
	}

	primaryUrl := os.Getenv("TURSO_DATABASE_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")

	// Log the connection details (for debugging)
	// log.Printf("Connecting to database URL: %s", primaryUrl)

	// Construct the connection string
	connectionString := fmt.Sprintf("%s?authToken=%s",
		strings.TrimSpace(primaryUrl),
		url.QueryEscape(authToken),
	)

	// Open the database connection
	db, err := sql.Open("libsql", connectionString)
	if err != nil {
		return nil, fmt.Errorf("error preparing database connection: %v", err)
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close() // Ensure we close the connection if ping fails
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
