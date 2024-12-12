// internal/representatives.go
package internal

import "database/sql"

type Representative struct {
	ID     int64
	Name   string
	Active bool
}

func LoadRepresentatives(db *sql.DB) ([]Representative, error) {
	rows, err := db.Query(`
        SELECT id, name, active
        FROM representatives
        WHERE active = true
        ORDER BY name
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var representatives []Representative
	for rows.Next() {
		var r Representative
		err := rows.Scan(&r.ID, &r.Name, &r.Active)
		if err != nil {
			return nil, err
		}
		representatives = append(representatives, r)
	}
	return representatives, nil
}
