// internal/products.go
package internal

import (
	"database/sql"
)

type Product struct {
	ID     int64
	Name   string
	Price  float64
	Active bool
}

func LoadProducts(db *sql.DB) ([]Product, error) {
	rows, err := db.Query(
		`SELECT id, name, price, active
    FROM products
    WHERE active = true
    ORDER BY name
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Active)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}
