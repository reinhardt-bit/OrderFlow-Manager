package internal

import (
	"database/sql"
)

type Product struct {
    ID        int64
    Name      string
    BasePrice float64
}

type Flavor struct {
    ID        int64
    Name      string
    ProductID int64
}

func LoadProducts(db *sql.DB) ([]Product, error) {
    rows, err := db.Query("SELECT id, name, base_price FROM products ORDER BY name")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var products []Product
    for rows.Next() {
        var p Product
        err := rows.Scan(&p.ID, &p.Name, &p.BasePrice)
        if err != nil {
            return nil, err
        }
        products = append(products, p)
    }
    return products, nil
}

func LoadFlavors(db *sql.DB, productID int64) ([]Flavor, error) {
    rows, err := db.Query("SELECT id, name, product_id FROM flavors WHERE product_id = ? ORDER BY name", productID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var flavors []Flavor
    for rows.Next() {
        var f Flavor
        err := rows.Scan(&f.ID, &f.Name, &f.ProductID)
        if err != nil {
            return nil, err
        }
        flavors = append(flavors, f)
    }
    return flavors, nil
}
