// internal/loadOrders.go
package internal

import (
	"database/sql"
	"time"
)

type Order struct {
	ID                 int64
	CreatedAt          time.Time
	ClientName         string
	Contact            string
	ProductID          int64
	ProductName        string
	RepresentativeID   int64
	RepresentativeName string
	Quantity           int
	Price              float64
	NeedsDelivery      bool
	DeliveryAddress    string
	Comment            string
	Completed          bool
}

func LoadOrders(db *sql.DB) ([]Order, error) {
	rows, err := db.Query(`
        SELECT o.id, o.created_at, o.client_name, o.contact,
               o.product_id, p.name, o.representative_id, r.name,
               o.quantity, o.price, o.needs_delivery, o.delivery_address,
               o.comment, o.completed
        FROM orders o
        JOIN products p ON o.product_id = p.id
        LEFT JOIN representatives r ON o.representative_id = r.id
        WHERE o.completed = false
        ORDER BY o.created_at DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		err := rows.Scan(
			&o.ID, &o.CreatedAt, &o.ClientName, &o.Contact,
			&o.ProductID, &o.ProductName, &o.RepresentativeID, &o.RepresentativeName,
			&o.Quantity, &o.Price, &o.NeedsDelivery, &o.DeliveryAddress,
			&o.Comment, &o.Completed,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}
