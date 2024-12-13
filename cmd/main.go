// cmd/main.go
package main

import (
	"blissfulBytes-manager/internal"
	"blissfulBytes-manager/shared/db"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/xuri/excelize/v2"
)

func main() {
	// myApp := app.New()
	myApp := app.NewWithID("com.blissfulbytes.manager")
	myWindow := myApp.NewWindow("Blissful Bites Manager")

	// Update environment variables from config file
	if err := db.UpdateEnvForDbConfig(); err != nil {
		log.Printf("Error updating database config: %v", err)
	}

	// Validate database configuration
	var database *sql.DB
	var dbErr error

	if err := db.ValidateDbConfig(); err != nil {
		log.Printf("Database configuration validation failed: %v", err)

		// Show database configuration dialog
		showDatabaseConfigDialog(myWindow, func() {
			// Attempt to initialize DB after configuration
			database, dbErr = db.InitDB()
			if dbErr != nil {
				dialog.ShowError(dbErr, myWindow)
				return
			}
			// Continue with app initialization
			// initializeMainApp(myApp, myWindow, database)
			initializeMainApp(myWindow, database)
		})
	} else {
		// Configuration is valid, proceed normally
		database, dbErr = db.InitDB()
		if dbErr != nil {
			dialog.ShowError(dbErr, myWindow)
			return
		}

		// Continue with app initialization
		// initializeMainApp(myApp, myWindow, database)
		initializeMainApp(myWindow, database)
	}

	myWindow.ShowAndRun()
}

func showAddProductDialog(window fyne.Window, db *sql.DB) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Product Name")

	priceEntry := widget.NewEntry()
	priceEntry.SetPlaceHolder("Price")

	content := container.NewVBox(
		nameEntry,
		priceEntry,
	)

	dialog := dialog.NewCustomConfirm(
		"Add New Product",
		"Add",
		"Cancel",
		content,
		func(submit bool) {
			if !submit {
				return
			}

			price, err := strconv.ParseFloat(priceEntry.Text, 64)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Invalid price"), window)
				return
			}

			_, err = db.Exec("INSERT INTO products (name, price, active) VALUES (?, ?, true)",
				nameEntry.Text, price)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			dialog.ShowInformation("Success", "Product added successfully", window)
		},
		window,
	)

	dialog.Show()
}

func showManageProductsDialog(window fyne.Window, db *sql.DB) {
	products, err := internal.LoadProducts(db)
	if err != nil {
		dialog.ShowError(err, window)
		return
	}

	list := widget.NewTable(
		func() (int, int) {
			return len(products), 1
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Template"),
				widget.NewButton("Edit", func() {}),
				widget.NewButton("Deactivate", func() {}),
			)
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			box := cell.(*fyne.Container)
			label := box.Objects[0].(*widget.Label)
			editBtn := box.Objects[1].(*widget.Button)
			deactivateBtn := box.Objects[2].(*widget.Button)

			product := products[id.Row]
			label.SetText(fmt.Sprintf("%s - R%.2f", product.Name, product.Price))

			editBtn.OnTapped = func() {
				showEditProductDialog(window, db, product)
			}

			deactivateBtn.OnTapped = func() {
				dialog.ShowConfirm("Deactivate Product",
					"Are you sure you want to deactivate this product? It will no longer be available for new orders.",
					func(confirm bool) {
						if confirm {
							_, err := db.Exec("UPDATE products SET active = false WHERE id = ?", product.ID)
							if err != nil {
								dialog.ShowError(err, window)
								return
							}
							showManageProductsDialog(window, db)
						}
					},
					window,
				)
			}
		},
	)

	list.SetColumnWidth(0, 500)

	content := container.NewVScroll(list)
	content.Resize(fyne.NewSize(600, 400))

	dialog := dialog.NewCustom("Manage Products", "Close", content, window)
	dialog.Resize(fyne.NewSize(600, 400))
	dialog.Show()
}

func showEditProductDialog(window fyne.Window, db *sql.DB, product internal.Product) {
	nameEntry := widget.NewEntry()
	nameEntry.SetText(product.Name)

	priceEntry := widget.NewEntry()
	priceEntry.SetText(fmt.Sprintf("%.2f", product.Price))

	content := container.NewVBox(
		nameEntry,
		priceEntry,
	)

	dialog := dialog.NewCustomConfirm(
		"Edit Product",
		"Save",
		"Cancel",
		content,
		func(submit bool) {
			if !submit {
				return
			}

			price, err := strconv.ParseFloat(priceEntry.Text, 64)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Invalid price"), window)
				return
			}

			_, err = db.Exec("UPDATE products SET name = ?, price = ? WHERE id = ?",
				nameEntry.Text, price, product.ID)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			showManageProductsDialog(window, db)
		},
		window,
	)
	dialog.Resize(fyne.NewSize(400, 400))
	dialog.Show()
}

func showAddOrderDialog(window fyne.Window, db *sql.DB, refreshTable func()) {
	products, err := internal.LoadProducts(db)
	if err != nil {
		dialog.ShowError(err, window)
		return
	}

	var productNames []string
	for _, p := range products {
		productNames = append(productNames, p.Name)
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Client Name")

	contactEntry := widget.NewEntry()
	contactEntry.SetPlaceHolder("Contact")

	productSelect := widget.NewSelect(productNames, nil)
	productSelect.PlaceHolder = "Select product"

	quantityEntry := widget.NewEntry()
	quantityEntry.SetPlaceHolder("Quantity")

	priceLabel := widget.NewLabel("Price: R0.00")
	var selectedProduct internal.Product

	productSelect.OnChanged = func(value string) {
		for _, p := range products {
			if p.Name == value {
				selectedProduct = p
				priceLabel.SetText(fmt.Sprintf("Price: R%.2f", p.Price))
				break
			}
		}
	}

	deliveryCheck := widget.NewCheck("Needs Delivery", nil)

	addressEntry := widget.NewMultiLineEntry()
	addressEntry.SetPlaceHolder("Delivery Address")

	commentEntry := widget.NewMultiLineEntry()
	commentEntry.SetPlaceHolder("Comment")

	representatives, err := internal.LoadRepresentatives(db)
	if err != nil {
		dialog.ShowError(err, window)
		return
	}

	var repNames []string
	for _, r := range representatives {
		repNames = append(repNames, r.Name)
	}

	repSelect := widget.NewSelect(repNames, nil)
	repSelect.PlaceHolder = "Select rep"

	content := container.NewVBox(
		repSelect,
		nameEntry,
		contactEntry,
		productSelect,
		quantityEntry,
		priceLabel,
		deliveryCheck,
		addressEntry,
		commentEntry,
	)

	dialog := dialog.NewCustomConfirm(
		"Add New Order",
		"Add",
		"Cancel",
		content,
		func(submit bool) {
			if !submit {
				return
			}

			// Find selected representative ID
			var repID int64
			for _, r := range representatives {
				if r.Name == repSelect.Selected {
					repID = r.ID
					break
				}
			}

			quantity, err := strconv.Atoi(quantityEntry.Text)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Invalid quantity"), window)
				return
			}

			totalPrice := selectedProduct.Price * float64(quantity)

			_, err = db.Exec(`
                INSERT INTO orders (
                    created_at, client_name, contact, product_id,
                    representative_id, quantity, price, needs_delivery,
                    delivery_address, comment, completed
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				time.Now(), nameEntry.Text, contactEntry.Text,
				selectedProduct.ID, repID, quantity, totalPrice,
				deliveryCheck.Checked, addressEntry.Text,
				commentEntry.Text, false,
			)

			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			refreshTable()
		},
		window,
	)

	dialog.Resize(fyne.NewSize(600, 500))
	dialog.Show()
}

func exportTableToExcel(db *sql.DB, tableName, fileName string) error {
	// Query the table
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		return fmt.Errorf("error querying table: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("error fetching column names: %w", err)
	}

	// Create a new Excel file
	f := excelize.NewFile()
	sheetName := "Sheet1"
	f.SetSheetName("Sheet1", sheetName)

	// Write column headers
	for i, col := range columns {
		columnName, _ := excelize.ColumnNumberToName(i + 1)
		cell := columnName + "1"
		f.SetCellValue(sheetName, cell, col)
	}

	// Write rows
	rowIndex := 2
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("error scanning row: %w", err)
		}

		for i, val := range values {
			columnName, _ := excelize.ColumnNumberToName(i + 1)
			cell := columnName + fmt.Sprintf("%d", rowIndex)
			f.SetCellValue(sheetName, cell, val)
		}
		rowIndex++
	}

	// Save the Excel file
	if err := f.SaveAs(fileName); err != nil {
		return fmt.Errorf("error saving Excel file: %w", err)
	}

	return nil
}

func showAddRepresentativeDialog(window fyne.Window, db *sql.DB) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Representative Name")

	content := container.NewVBox(nameEntry)

	dialog := dialog.NewCustomConfirm(
		"Add New Representative",
		"Add",
		"Cancel",
		content,
		func(submit bool) {
			if !submit {
				return
			}

			_, err := db.Exec("INSERT INTO representatives (name, active) VALUES (?, true)",
				nameEntry.Text)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			dialog.ShowInformation("Success", "Representative added successfully", window)
		},
		window,
	)

	dialog.Show()
}

func showManageRepresentativesDialog(window fyne.Window, db *sql.DB) {
	representatives, err := internal.LoadRepresentatives(db)
	if err != nil {
		dialog.ShowError(err, window)
		return
	}

	list := widget.NewTable(
		func() (int, int) {
			return len(representatives), 1
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Template"),
				widget.NewButton("Deactivate", func() {}),
			)
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			box := cell.(*fyne.Container)
			label := box.Objects[0].(*widget.Label)
			deactivateBtn := box.Objects[1].(*widget.Button)

			rep := representatives[id.Row]
			label.SetText(rep.Name)

			deactivateBtn.OnTapped = func() {
				dialog.ShowConfirm("Deactivate Representative",
					"Are you sure you want to deactivate this representative?",
					func(confirm bool) {
						if confirm {
							_, err := db.Exec("UPDATE representatives SET active = false WHERE id = ?", rep.ID)
							if err != nil {
								dialog.ShowError(err, window)
								return
							}
							showManageRepresentativesDialog(window, db)
						}
					},
					window,
				)
			}
		},
	)

	list.SetColumnWidth(0, 400)

	content := container.NewVScroll(list)
	content.Resize(fyne.NewSize(500, 400))

	dialog := dialog.NewCustom("Manage Representatives", "Close", content, window)
	dialog.Resize(fyne.NewSize(500, 400))
	dialog.Show()
}

// Implement showDatabaseConfigDialog using the new db package methods
func showDatabaseConfigDialog(window fyne.Window, onSaveCallback func()) {
	// Load existing config
	existingConfig, _ := db.LoadDbConfig()

	// Create entries for database URL and auth token
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("Turso Database URL")
	urlEntry.SetText(existingConfig.DatabaseURL)

	tokenEntry := widget.NewEntry()
	tokenEntry.SetPlaceHolder("Turso Auth Token")
	tokenEntry.SetText(existingConfig.AuthToken)

	content := container.NewVBox(
		widget.NewLabel("Configure Turso Database Connection"),
		widget.NewLabel("Database URL:"),
		urlEntry,
		widget.NewLabel("Auth Token:"),
		tokenEntry,
	)

	dialog := dialog.NewCustomConfirm(
		"Database Configuration",
		"Save",
		"Cancel",
		content,
		func(submit bool) {
			if !submit {
				// Exit the application if user cancels configuration
				window.Close()
				return
			}

			// Create and save new configuration
			newConfig := db.DatabaseConfig{
				DatabaseURL: urlEntry.Text,
				AuthToken:   tokenEntry.Text,
			}

			err := db.SaveDbConfig(newConfig)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			dialog.ShowInformation("Success", "Database configuration saved", window)

			// Call the provided callback
			if onSaveCallback != nil {
				onSaveCallback()
			}
		},
		window,
	)

	dialog.Resize(fyne.NewSize(600, 300))
	dialog.Show()
}

// New function to initialize main app components
// func initializeMainApp(myApp fyne.App, myWindow fyne.Window, db *sql.DB) {
func initializeMainApp(myWindow fyne.Window, db *sql.DB) {
	// defer db.Close()

	// Create menu items
	mainMenu := fyne.NewMainMenu(
		fyne.NewMenu("Products",
			fyne.NewMenuItem("Add New Product", func() {
				showAddProductDialog(myWindow, db)
			}),
			fyne.NewMenuItem("Manage Products", func() {
				showManageProductsDialog(myWindow, db)
			}),
		),
		fyne.NewMenu("Representatives",
			fyne.NewMenuItem("Add New Representative", func() {
				showAddRepresentativeDialog(myWindow, db)
			}),
			fyne.NewMenuItem("Manage Representatives", func() {
				showManageRepresentativesDialog(myWindow, db)
			}),
		),
		fyne.NewMenu("Settings",
			fyne.NewMenuItem("Database Connection", func() {
				showDatabaseConfigDialog(myWindow, nil)
			}),
		),
	)

	// Set the main menu
	myWindow.SetMainMenu(mainMenu)

	// Initialize order table
	orderTable := widget.NewTable(
		func() (int, int) { return 6, 5 },
		func() fyne.CanvasObject {
			return widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{})
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {},
	)

	// Refresh function for the order table
	refreshTable := func() {
		orders, err := internal.LoadOrders(db)
		if err != nil {
			log.Printf("Error loading orders: %v", err)
			return
		}

		orderTable.Length = func() (int, int) {
			return len(orders) + 1, 5 // +1 for header row
		}

		orderTable.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
			orderTable.SetColumnWidth(0, 150) // Date
			orderTable.SetColumnWidth(1, 200) // Client
			orderTable.SetColumnWidth(2, 300) // Product
			orderTable.SetColumnWidth(3, 100) // Price
			orderTable.SetColumnWidth(4, 150) // Representative
			orderTable.SetColumnWidth(5, 50)  // Status

			label := cell.(*widget.Label)
			if id.Row == 0 {
				// Header row
				switch id.Col {
				case 0:
					label.SetText("Date")
				case 1:
					label.SetText("Client")
				case 2:
					label.SetText("Product")
				case 3:
					label.SetText("Total")
				case 4:
					label.SetText("Representative")
				case 5:
					label.SetText("Status")
				}
				return
			}

			order := orders[id.Row-1]
			switch id.Col {
			case 0:
				label.SetText(order.CreatedAt.Format("2006-01-02 15:04"))
			case 1:
				label.SetText(order.ClientName)
			case 2:
				label.SetText(fmt.Sprintf("%d x %s", order.Quantity, order.ProductName))
			case 3:
				label.SetText(fmt.Sprintf("R%.2f", order.Price))
			case 4:
				label.SetText(order.RepresentativeName)
			case 5:
				if order.Completed {
					label.SetText("Completed")
				} else {
					label.SetText("Pending")
				}
			}
		}
		orderTable.Refresh()
	}

	// Add new order button
	addOrderBtn := widget.NewButton("+", func() {
		showAddOrderDialog(myWindow, db, refreshTable)
	})

	// Create the layout
	form := container.NewVBox(
		widget.NewLabel("Orders"),
		addOrderBtn,
	)

	// Track selected row
	var selectedRow int = -1
	orderTable.OnSelected = func(id widget.TableCellID) {
		selectedRow = id.Row
	}

	// Complete order button
	completeBtn := widget.NewButton("Mark Selected as Completed", func() {
		if selectedRow > 0 {
			orders, _ := internal.LoadOrders(db)
			orderID := orders[selectedRow-1].ID

			_, err := db.Exec("UPDATE orders SET completed = true WHERE id = ?", orderID)
			if err != nil {
				log.Printf("Error completing order: %v", err)
				return
			}
			refreshTable()
		}
	})

	downloadOrdersBtn := widget.NewButton("Download Orders", func() {
		// Create dialog with file save picker
		dialog := dialog.NewFileSave(
			func(writer fyne.URIWriteCloser, err error) {
				if err != nil {
					dialog.ShowError(err, myWindow)
					return
				}
				if writer == nil {
					return // user cancelled
				}
				defer writer.Close()

				// Get the selected path and ensure it ends with .xlsx
				path := writer.URI().Path()
				if !strings.HasSuffix(strings.ToLower(path), ".xlsx") {
					path += ".xlsx"
				}

				// Export the orders
				if err := exportOrdersToExcel(db, path); err != nil {
					dialog.ShowError(err, myWindow)
					return
				}

				dialog.ShowInformation("Success",
					"Orders have been exported successfully to:\n"+path,
					myWindow)
			},
			myWindow)

		// Set default filename
		dialog.SetFileName(fmt.Sprintf("orders_%s.xlsx",
			time.Now().Format("2006-01-02")))

		// Set filter for Excel files
		dialog.SetFilter(storage.NewExtensionFileFilter([]string{".xlsx"}))

		dialog.Show()
	})

	split := container.NewHSplit(
		form,
		container.NewBorder(
			nil,
			container.NewHBox(
				completeBtn,
				downloadOrdersBtn,
			),
			// completeBtn,
			nil,
			nil,
			orderTable,
		),
	)
	split.SetOffset(0.03)

	myWindow.SetContent(split)
	myWindow.Resize(fyne.NewSize(1024, 768))

	// Initial table load
	refreshTable()

	// Add close handler
	myWindow.SetOnClosed(func() {
		db.Close()
	})

	// myWindow.ShowAndRun()
}

func exportOrdersToExcel(db *sql.DB, filePath string) error {
	// Query orders with joined product and representative information
	query := `
        SELECT
            o.id,
            o.created_at,
            o.client_name,
            o.contact,
            p.name as product_name,
            o.quantity,
            o.price,
            o.needs_delivery,
            o.delivery_address,
            o.comment,
            o.completed,
            r.name as representative_name
        FROM orders o
        LEFT JOIN products p ON o.product_id = p.id
        LEFT JOIN representatives r ON o.representative_id = r.id
        ORDER BY o.created_at DESC
    `

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("error querying orders: %w", err)
	}
	defer rows.Close()

	// Create a new Excel file
	f := excelize.NewFile()
	sheetName := "Orders"
	f.SetSheetName("Sheet1", sheetName)

	// Define headers
	headers := []string{
		"Order ID",
		"Date",
		"Client Name",
		"Contact",
		"Product",
		"Quantity",
		"Total Price",
		"Needs Delivery",
		"Delivery Address",
		"Comment",
		"Status",
		"Representative",
	}

	// Write headers
	for i, header := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetCellValue(sheetName, col+"1", header)

		// Set column width based on content
		f.SetColWidth(sheetName, col, col, 15)
	}

	// Write data rows
	rowIndex := 2
	for rows.Next() {
		var (
			id            int64
			createdAt     time.Time
			clientName    string
			contact       string
			productName   string
			quantity      int
			price         float64
			needsDelivery bool
			deliveryAddr  sql.NullString
			comment       sql.NullString
			completed     bool
			repName       sql.NullString
		)

		err := rows.Scan(
			&id,
			&createdAt,
			&clientName,
			&contact,
			&productName,
			&quantity,
			&price,
			&needsDelivery,
			&deliveryAddr,
			&comment,
			&completed,
			&repName,
		)
		if err != nil {
			return fmt.Errorf("error scanning row: %w", err)
		}

		// Format status
		status := "Pending"
		if completed {
			status = "Completed"
		}

		// Write row data
		rowData := []interface{}{
			id,
			createdAt.Format("2006-01-02 15:04"),
			clientName,
			contact,
			productName,
			quantity,
			fmt.Sprintf("R%.2f", price),
			needsDelivery,
			deliveryAddr.String,
			comment.String,
			status,
			repName.String,
		}

		for i, value := range rowData {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetCellValue(sheetName, fmt.Sprintf("%s%d", col, rowIndex), value)
		}
		rowIndex++
	}

	// Apply some styling
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
	})
	if err == nil {
		// Apply style to header row
		f.SetRowStyle(sheetName, 1, 1, style)
	}

	// Auto-filter for all columns
	lastCol, _ := excelize.ColumnNumberToName(len(headers))
	// f.AutoFilter(sheetName, "A1", fmt.Sprintf("%s%d", lastCol, rowIndex-1), nil)
	ref := fmt.Sprintf("A1:%s%d", lastCol, rowIndex-1)
	f.AutoFilter(sheetName, ref, []excelize.AutoFilterOptions{})

	// Save the file
	if err := f.SaveAs(filePath); err != nil {
		return fmt.Errorf("error saving Excel file: %w", err)
	}

	return nil
}
