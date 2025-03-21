// internal/products_test.go
package internal

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3" // Use SQLite for testing
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create test tables
	_, err = db.Exec(`
		CREATE TABLE products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			price REAL NOT NULL,
			active BOOLEAN DEFAULT true
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

func TestLoadProducts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO products (name, price, active) VALUES
		('Test Product 1', 10.99, true),
		('Test Product 2', 20.99, true),
		('Inactive Product', 15.99, false)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Call the function being tested
	products, err := LoadProducts(db)
	if err != nil {
		t.Fatalf("LoadProducts failed: %v", err)
	}

	// Verify results
	if len(products) != 2 {
		t.Errorf("Expected 2 active products, got %d", len(products))
	}

	if products[0].Name != "Test Product 1" || products[0].Price != 10.99 {
		t.Errorf("First product data incorrect: %+v", products[0])
	}

	if products[1].Name != "Test Product 2" || products[1].Price != 20.99 {
		t.Errorf("Second product data incorrect: %+v", products[1])
	}
}
