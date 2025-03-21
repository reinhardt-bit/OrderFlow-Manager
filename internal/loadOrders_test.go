package internal

import (
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLoadOrders(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Set up expected results
	now := time.Now()
	dueDate := now.AddDate(0, 0, 7)

	// Expected orders query
	mock.ExpectQuery("SELECT o.id, o.created_at, o.due_date, o.client_name, o.contact, o.representative_id, r.name, o.needs_delivery, o.delivery_address, o.comment, o.completed, o.total_price FROM orders o LEFT JOIN representatives r ON o.representative_id = r.id WHERE o.completed = false ORDER BY o.created_at DESC").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "due_date", "client_name", "contact",
			"representative_id", "rep_name", "needs_delivery", "delivery_address",
			"comment", "completed", "total_price",
		}).
		AddRow(1, now, dueDate, "Test Client", "123-456-7890",
			2, "John Doe", false, "",
			"Test comment", false, 25.50))

	// Expected order items query
	mock.ExpectQuery("SELECT oi.id, oi.product_id, p.name, oi.quantity, oi.price FROM order_items oi JOIN products p ON oi.product_id = p.id WHERE oi.order_id = ?").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "product_id", "name", "quantity", "price",
		}).
		AddRow(1, 1, "Test Product", 2, 25.50))

	// Call the function being tested
	orders, err := LoadOrders(db)
	if err != nil {
		t.Fatalf("Failed to load orders: %v", err)
	}

	// Check expectations
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}

	// Validate results
	if len(orders) != 1 {
		t.Fatalf("Expected 1 order, got %d", len(orders))
	}

	// Verify the order details
	order := orders[0]
	if order.ID != 1 {
		t.Errorf("Expected order ID 1, got %d", order.ID)
	}
	if order.ClientName != "Test Client" {
		t.Errorf("Expected client name 'Test Client', got '%s'", order.ClientName)
	}
	if order.RepresentativeName != "John Doe" {
		t.Errorf("Expected representative name 'John Doe', got '%s'", order.RepresentativeName)
	}
	if order.TotalPrice != 25.50 {
		t.Errorf("Expected total price 25.50, got %.2f", order.TotalPrice)
	}

	// Verify the order items
	if len(order.Items) != 1 {
		t.Fatalf("Expected 1 order item, got %d", len(order.Items))
	}

	item := order.Items[0]
	if item.ProductID != 1 {
		t.Errorf("Expected product ID 1, got %d", item.ProductID)
	}
	if item.ProductName != "Test Product" {
		t.Errorf("Expected product name 'Test Product', got '%s'", item.ProductName)
	}
	if item.Quantity != 2 {
		t.Errorf("Expected quantity 2, got %d", item.Quantity)
	}
	if item.Price != 25.50 {
		t.Errorf("Expected price 25.50, got %.2f", item.Price)
	}
}

func TestLoadOrders_NoOrders(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Expect query but return empty result
	mock.ExpectQuery("SELECT o.id, o.created_at, o.due_date, o.client_name, o.contact, o.representative_id, r.name, o.needs_delivery, o.delivery_address, o.comment, o.completed, o.total_price FROM orders o LEFT JOIN representatives r ON o.representative_id = r.id WHERE o.completed = false ORDER BY o.created_at DESC").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "due_date", "client_name", "contact",
			"representative_id", "rep_name", "needs_delivery", "delivery_address",
			"comment", "completed", "total_price",
		}))

	// Call the function being tested
	orders, err := LoadOrders(db)
	if err != nil {
		t.Fatalf("Failed to load orders: %v", err)
	}

	// Check expectations
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}

	// Validate results
	if len(orders) != 0 {
		t.Fatalf("Expected 0 orders, got %d", len(orders))
	}
}

func TestEditOrder(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Sample order for testing
	dueDate := time.Now().AddDate(0, 0, 7)
	order := Order{
		ID:               1,
		DueDate:          dueDate,
		ClientName:       "Updated Client",
		Contact:          "987-654-3210",
		RepresentativeID: 3,
		NeedsDelivery:    true,
		DeliveryAddress:  "123 Main St",
		Comment:          "Updated comment",
		TotalPrice:       35.75,
		Items: []OrderItem{
			{
				ProductID: 2,
				Quantity:  3,
				Price:     35.75,
			},
		},
	}

	// Expect transaction to begin
	mock.ExpectBegin()

	// Expect update query
	mock.ExpectExec("UPDATE orders SET due_date = \\?, client_name = \\?, contact = \\?, representative_id = \\?, needs_delivery = \\?, delivery_address = \\?, comment = \\?, total_price = \\? WHERE id = \\?").
		WithArgs(
			order.DueDate,
			order.ClientName,
			order.Contact,
			order.RepresentativeID,
			order.NeedsDelivery,
			order.DeliveryAddress,
			order.Comment,
			order.TotalPrice,
			order.ID,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect delete of existing items
	mock.ExpectExec("DELETE FROM order_items WHERE order_id = \\?").
		WithArgs(order.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect insert of new items
	mock.ExpectExec("INSERT INTO order_items \\(order_id, product_id, quantity, price\\) VALUES \\(\\?, \\?, \\?, \\?\\)").
		WithArgs(order.ID, order.Items[0].ProductID, order.Items[0].Quantity, order.Items[0].Price).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect commit
	mock.ExpectCommit()

	// Call the function being tested
	err = EditOrder(db, order)
	if err != nil {
		t.Fatalf("Failed to edit order: %v", err)
	}

	// Check expectations
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestEditOrder_TransactionError(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Sample order for testing
	order := Order{
		ID:         1,
		DueDate:    time.Now(),
		ClientName: "Test Client",
	}

	// Expect transaction to begin but fail
	mock.ExpectBegin().WillReturnError(fmt.Errorf("transaction error"))

	// Call the function being tested
	err = EditOrder(db, order)

	// Should return an error
	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	// Check expectations
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestEditOrder_UpdateError(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Sample order for testing
	order := Order{
		ID:         1,
		DueDate:    time.Now(),
		ClientName: "Test Client",
	}

	// Expect transaction to begin
	mock.ExpectBegin()

	// Expect update query to fail
	mock.ExpectExec("UPDATE orders").
		WithArgs(
			order.DueDate,
			order.ClientName,
			order.Contact,
			order.RepresentativeID,
			order.NeedsDelivery,
			order.DeliveryAddress,
			order.Comment,
			order.TotalPrice,
			order.ID,
		).
		WillReturnError(fmt.Errorf("update error"))

	// Expect rollback due to error
	mock.ExpectRollback()

	// Call the function being tested
	err = EditOrder(db, order)

	// Should return an error
	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	// Check if the error message contains "update error"
	if err.Error() != "update error" {
		t.Errorf("Expected error message 'update error', got '%s'", err.Error())
	}

	// Check expectations
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}
