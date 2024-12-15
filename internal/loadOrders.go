// internal/loadOrders.go
package internal

import (
	"database/sql"
	"time"
)

type OrderItem struct {
	ID          int64
	ProductID   int64
	ProductName string
	Quantity    int
	Price       float64
}

type Order struct {
	ID                 int64
	CreatedAt          time.Time
	DueDate            time.Time
	ClientName         string
	Contact            string
	RepresentativeID   int64
	RepresentativeName string
	NeedsDelivery      bool
	DeliveryAddress    string
	Comment            string
	Completed          bool
	TotalPrice         float64
	Items              []OrderItem
}

func LoadOrders(db *sql.DB) ([]Order, error) {
	rows, err := db.Query(`
        SELECT o.id, o.created_at, o.due_date, o.client_name, o.contact,
               o.representative_id, r.name, o.needs_delivery, o.delivery_address,
               o.comment, o.completed, o.total_price
        FROM orders o
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
			&o.ID, &o.CreatedAt, &o.DueDate, &o.ClientName, &o.Contact,
			&o.RepresentativeID, &o.RepresentativeName, &o.NeedsDelivery,
			&o.DeliveryAddress, &o.Comment, &o.Completed, &o.TotalPrice,
		)
		if err != nil {
			return nil, err
		}

		// Load order items
		itemRows, err := db.Query(`
            SELECT oi.id, oi.product_id, p.name, oi.quantity, oi.price
            FROM order_items oi
            JOIN products p ON oi.product_id = p.id
            WHERE oi.order_id = ?
        `, o.ID)
		if err != nil {
			return nil, err
		}
		defer itemRows.Close()

		for itemRows.Next() {
			var item OrderItem
			err := itemRows.Scan(&item.ID, &item.ProductID, &item.ProductName,
				&item.Quantity, &item.Price)
			if err != nil {
				return nil, err
			}
			o.Items = append(o.Items, item)
		}

		orders = append(orders, o)
	}
	return orders, nil
}

func EditOrder(db *sql.DB, order Order) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update main order
	_, err = tx.Exec(`
        UPDATE orders
        SET due_date = ?, client_name = ?, contact = ?,
            representative_id = ?, needs_delivery = ?,
            delivery_address = ?, comment = ?, total_price = ?
        WHERE id = ?`,
		order.DueDate, order.ClientName, order.Contact,
		order.RepresentativeID, order.NeedsDelivery,
		order.DeliveryAddress, order.Comment, order.TotalPrice,
		order.ID)
	if err != nil {
		return err
	}

	// Delete existing order items
	_, err = tx.Exec("DELETE FROM order_items WHERE order_id = ?", order.ID)
	if err != nil {
		return err
	}

	// Insert new order items
	for _, item := range order.Items {
		_, err = tx.Exec(`
            INSERT INTO order_items (order_id, product_id, quantity, price)
            VALUES (?, ?, ?, ?)`,
			order.ID, item.ProductID, item.Quantity, item.Price)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
