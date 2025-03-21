// internal/representatives_test.go
package internal

import (
	"testing"
)

func TestLoadRepresentatives(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create representatives table
	_, err := db.Exec(`
		CREATE TABLE representatives (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			active BOOLEAN DEFAULT true
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create representatives table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO representatives (name, active) VALUES
		('John Doe', true),
		('Jane Smith', true),
		('Inactive Rep', false)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Call the function being tested
	reps, err := LoadRepresentatives(db)
	if err != nil {
		t.Fatalf("LoadRepresentatives failed: %v", err)
	}

	// Verify results
	if len(reps) != 2 {
		t.Errorf("Expected 2 active representatives, got %d", len(reps))
	}

	if reps[0].Name != "Jane Smith" { // Alphabetical order
		t.Errorf("First representative data incorrect: %+v", reps[0])
	}

	if reps[1].Name != "John Doe" {
		t.Errorf("Second representative data incorrect: %+v", reps[1])
	}
}
