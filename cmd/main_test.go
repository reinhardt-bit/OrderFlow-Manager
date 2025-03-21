// cmd/main_test.go
package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/xuri/excelize/v2"
)

func TestExportOrdersToExcel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Mock data
	rows := sqlmock.NewRows([]string{
		"id", "representative_name", "completed", "created_at", "client_name",
		"contact", "due_date", "product_name", "quantity", "item_price",
		"product_price", "total_price", "comment",
	}).AddRow(
		1,
		"John Doe",
		true,
		time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		"Client A",
		"client@example.com",
		time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC),
		"Product X",
		2,
		15.0,
		30.0,
		30.0,
		"Urgent order",
	)

	mock.ExpectQuery(`SELECT`).WillReturnRows(rows)

	// Temporary file path
	tmpFile := filepath.Join(t.TempDir(), "orders_test.xlsx")

	// Execute export
	err = exportOrdersToExcel(db, tmpFile)
	if err != nil {
		t.Fatalf("exportOrdersToExcel failed: %v", err)
	}

	// Verify Excel content
	f, err := excelize.OpenFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to open Excel file: %v", err)
	}

	// Validate headers
	expectedHeaders := []string{
		"Order ID", "Representative", "Status", "Date",
		"Client Name", "Contact", "Due Date", "Product Name",
		"Product Quantity", "Product Unit Price", "Product Total",
		"Total Order Price", "Comment",
	}

	sheetRows, err := f.GetRows("Orders")
	if err != nil {
		t.Fatalf("failed to get sheet rows: %v", err)
	}

	if len(sheetRows) < 2 {
		t.Fatal("expected at least 2 rows (header + data)")
	}

	if !reflect.DeepEqual(sheetRows[0], expectedHeaders) {
		t.Errorf("expected headers %v, got %v", expectedHeaders, sheetRows[0])
	}

	// Validate data row
	expectedData := []string{
		"1",
		"John Doe",
		"Completed",
		"2023-01-01 12:00",
		"Client A",
		"client@example.com",
		"2023-01-05",
		"Product X",
		"2",
		"R15.00",
		"R30.00",
		"R30.00",
		"Urgent order",
	}

	if !reflect.DeepEqual(sheetRows[1], expectedData) {
		t.Errorf("expected data row %v, got %v", expectedData, sheetRows[1])
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestExportOrdersToExcel_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT`).WillReturnError(fmt.Errorf("mock database error"))

	tmpFile := filepath.Join(t.TempDir(), "orders_error_test.xlsx")

	err = exportOrdersToExcel(db, tmpFile)
	if err == nil || !strings.Contains(err.Error(), "mock database error") {
		t.Errorf("expected database error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestExportOrdersToExcel_EmptyData(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"id", "representative_name", "completed", "created_at", "client_name",
		"contact", "due_date", "product_name", "quantity", "item_price",
		"product_price", "total_price", "comment",
	})

	mock.ExpectQuery(`SELECT`).WillReturnRows(rows)

	tmpFile := filepath.Join(t.TempDir(), "orders_empty_test.xlsx")

	err = exportOrdersToExcel(db, tmpFile)
	if err != nil {
		t.Fatalf("exportOrdersToExcel failed: %v", err)
	}

	f, err := excelize.OpenFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to open Excel file: %v", err)
	}

	sheetRows, err := f.GetRows("Orders")
	if err != nil {
		t.Fatalf("failed to get sheet rows: %v", err)
	}

	if len(sheetRows) != 1 {
		t.Fatalf("expected only header row, got %d rows", len(sheetRows))
	}
}
