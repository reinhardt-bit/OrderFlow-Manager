package internal

import (
	"database/sql"
	"fmt"
	"time"
)

type Order struct {
	ID int64
	// CreatedAt       string
	CreatedAt       time.Time
	ClientName      string
	Contact         string
	Quantity        int
	Price           float64
	NeedsDelivery   bool
	DeliveryAddress string
	Completed       bool
}

func LoadOrders(db *sql.DB) ([]Order, error) {
	// Modified query to use correct column name
	rows, err := db.Query(`
        SELECT id, created_at, client_name, contact, quantity,
               price, needs_delivery, delivery_address, completed
        FROM orders
        WHERE completed = false
        ORDER BY created_at DESC
    `)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL: %v", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		err := rows.Scan(&o.ID, &o.CreatedAt, &o.ClientName, &o.Contact,
			&o.Quantity, &o.Price, &o.NeedsDelivery,
			&o.DeliveryAddress, &o.Completed)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}
		orders = append(orders, o)
	}

	// for _, order := range orders {
	// 	fmt.Println(order)
	// }
	return orders, nil
}
