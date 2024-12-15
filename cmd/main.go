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
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Client Name")

	contactEntry := widget.NewEntry()
	contactEntry.SetPlaceHolder("Contact")

	dueDatePicker := widget.NewEntry()
	dueDatePicker.SetPlaceHolder("Due Date (YYYY-MM-DD)")

	var orderItems []internal.OrderItem

	updateTotalPrice := func() float64 {
		var total float64
		for _, item := range orderItems {
			total += item.Price
		}
		return total
	}

	itemsButton := widget.NewButton("Manage Items", func() {
		products, err := internal.LoadProducts(db)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}
		showOrderItemsDialog(window, products, orderItems, func(items []internal.OrderItem) {
			orderItems = items
		})
	})

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
		dueDatePicker,
		itemsButton,
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

			dueDate, err := time.Parse("2006-01-02", dueDatePicker.Text)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Invalid due date format. Please use YYYY-MM-DD"), window)
				return
			}

			var repID int64
			for _, r := range representatives {
				if r.Name == repSelect.Selected {
					repID = r.ID
					break
				}
			}

			// Begin transaction
			tx, err := db.Begin()
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			defer tx.Rollback()

			// Insert main order
			result, err := tx.Exec(`
                INSERT INTO orders (
                    created_at, due_date, client_name, contact,
                    representative_id, comment, completed, total_price
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				time.Now(), dueDate, nameEntry.Text, contactEntry.Text,
				repID, commentEntry.Text, false, updateTotalPrice(),
			)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			orderID, err := result.LastInsertId()
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			// Insert order items
			for _, item := range orderItems {
				_, err = tx.Exec(`
                    INSERT INTO order_items (
                        order_id, product_id, quantity, price
                    ) VALUES (?, ?, ?, ?)`,
					orderID, item.ProductID, item.Quantity, item.Price,
				)
				if err != nil {
					dialog.ShowError(err, window)
					return
				}
			}

			if err := tx.Commit(); err != nil {
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
func initializeMainApp(myWindow fyne.Window, db *sql.DB) {

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
		func() (int, int) { return 6, 8 },
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
			return len(orders) + 1, 8 // +1 for header row
		}

		orderTable.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
			orderTable.SetColumnWidth(0, 150) // Date
			orderTable.SetColumnWidth(1, 200) // Client
			orderTable.SetColumnWidth(2, 300) // Products
			orderTable.SetColumnWidth(3, 100) // Total Price
			orderTable.SetColumnWidth(4, 150) // Representative
			orderTable.SetColumnWidth(5, 90)  // Due Date
			orderTable.SetColumnWidth(6, 80)  // Status
			orderTable.SetColumnWidth(7, 300) // Comment

			label := cell.(*widget.Label)
			label.Wrapping = fyne.TextWrapWord

			if id.Row == 0 {
				// Header row
				switch id.Col {
				case 0:
					label.SetText("Date")
				case 1:
					label.SetText("Client")
				case 2:
					label.SetText("Products")
				case 3:
					label.SetText("Total")
				case 4:
					label.SetText("Representative")
				case 5:
					label.SetText("Due Date")
				case 6:
					label.SetText("Status")
				case 7:
					label.SetText("Comment")
				}
				return
			} else {
				order := orders[id.Row-1]
				switch id.Col {
				case 0:
					label.SetText(order.CreatedAt.Format("2006-01-02 15:04"))
				case 1:
					clientInfo := fmt.Sprintf("%s\n%s", order.ClientName, order.Contact)
					label.SetText(clientInfo)
				case 2:
					var products []string
					for _, item := range order.Items {
						products = append(products, fmt.Sprintf("%d x %s", item.Quantity, item.ProductName))
					}
					label.SetText(strings.Join(products, "\n"))
					// Set minimum height based on number of products
					minHeight := 40 * float32(len(products))
					if minHeight < 45 {
						minHeight = 45
					}
					orderTable.SetRowHeight(id.Row, minHeight)
				case 3:
					label.SetText(fmt.Sprintf("R%.2f", order.TotalPrice))
				case 4:
					label.SetText(order.RepresentativeName)
				case 5:
					label.SetText(order.DueDate.Format("2006-01-02"))
				case 6:
					if order.Completed {
						label.SetText("Completed")
					} else {
						label.SetText("Pending")
					}
				case 7:
					label.SetText(order.Comment)
				}
			}
		}
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

	completeBtn := widget.NewButton("Mark Complete", func() {})

	split := container.NewHSplit(
		form,
		container.NewBorder(
			nil,
			container.NewHBox(
				completeBtn,
				downloadOrdersBtn,
			),
			nil,
			nil,
			orderTable,
		),
	)
	split.SetOffset(0.03)

	myWindow.SetContent(split)
	orderTable.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 {
			orders, _ := internal.LoadOrders(db)
			order := orders[id.Row-1]

			editBtn := widget.NewButton("Edit", func() {
				showEditOrderDialog(myWindow, db, order, refreshTable)
			})

			completeBtn.OnTapped = func() {
				_, err := db.Exec("UPDATE orders SET completed = true WHERE id = ?", order.ID)
				if err != nil {
					dialog.ShowError(err, myWindow)
					return
				}
				refreshTable()
			}

			actions := container.NewHBox(
				editBtn,
				completeBtn,
				downloadOrdersBtn,
			)

			content := container.NewHSplit(
				form,
				container.NewBorder(
					nil,
					actions,
					nil,
					nil,
					orderTable,
				),
			)

			content.SetOffset(0.03)
			myWindow.SetContent(content)
		}
	}

	myWindow.Resize(fyne.NewSize(1024, 768))

	// Initial table load
	refreshTable()

	// Add close handler
	myWindow.SetOnClosed(func() {
		db.Close()
	})
}

func exportOrdersToExcel(db *sql.DB, filePath string) error {
	// Query orders with joined product and representative information
	query := `
        SELECT
            o.id,
            r.name as representative_name,
            o.completed,
            o.created_at,
            o.client_name,
            o.contact,
            o.due_date,
            p.name as product_name,
            oi.quantity,
            p.price as product_price,
            oi.price as item_price,
            o.total_price,
            o.comment
        FROM orders o
        LEFT JOIN representatives r ON o.representative_id = r.id
        LEFT JOIN order_items oi ON o.id = oi.order_id
        LEFT JOIN products p ON oi.product_id = p.id
        ORDER BY o.created_at DESC, o.id, p.name
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
		"Representative",
		"Status",
		"Date",
		"Client Name",
		"Contact",
		"Due Date",
		"Product Name",
		"Product Quantity",
		"Product Unit Price",
		"Product Total",
		"Total Order Price",
		"Comment",
	}

	// Write headers
	for i, header := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetCellValue(sheetName, col+"1", header)

		// Set column width based on content
		f.SetColWidth(sheetName, col, col, 13)
	}

	// Write data rows
	rowIndex := 2
	for rows.Next() {
		var (
			id           int64
			repName      sql.NullString
			completed    bool
			createdAt    time.Time
			clientName   string
			contact      string
			dueDate      time.Time
			productName  sql.NullString
			quantity     sql.NullInt64
			itemPrice    sql.NullFloat64
			productPrice sql.NullFloat64
			totalPrice   float64
			comment      sql.NullString
		)

		err := rows.Scan(
			&id,
			&repName,
			&completed,
			&createdAt,
			&clientName,
			&contact,
			&dueDate,
			&productName,
			&quantity,
			&itemPrice,
			&productPrice,
			&totalPrice,
			&comment,
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
			repName.String,
			status,
			createdAt.Format("2006-01-02 15:04"),
			clientName,
			contact,
			dueDate.Format("2006-01-02"),
			productName.String,
			quantity.Int64,
			fmt.Sprintf("R%.2f", itemPrice.Float64),
			fmt.Sprintf("R%.2f", productPrice.Float64),
			fmt.Sprintf("R%.2f", totalPrice),
			comment.String,
		}

		for i, value := range rowData {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetCellValue(sheetName, fmt.Sprintf("%s%d", col, rowIndex), value)
		}
		rowIndex++
	}

	// Apply styling
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
	ref := fmt.Sprintf("A1:%s%d", lastCol, rowIndex-1)
	f.AutoFilter(sheetName, ref, []excelize.AutoFilterOptions{})

	// Save the file
	if err := f.SaveAs(filePath); err != nil {
		return fmt.Errorf("error saving Excel file: %w", err)
	}

	return nil
}

type OrderItemEntry struct {
	ProductSelect *widget.Select
	QuantityEntry *widget.Entry
	PriceLabel    *widget.Label
	DeleteButton  *widget.Button
}

func showEditOrderDialog(window fyne.Window, db *sql.DB, order internal.Order, refreshTable func()) {
	nameEntry := widget.NewEntry()
	nameEntry.SetText(order.ClientName)

	contactEntry := widget.NewEntry()
	contactEntry.SetText(order.Contact)

	dueDatePicker := widget.NewEntry()
	dueDatePicker.SetText(order.DueDate.Format("2006-01-02"))

	commentEntry := widget.NewMultiLineEntry()
	commentEntry.SetText(order.Comment)

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
	for _, r := range representatives {
		if r.ID == order.RepresentativeID {
			repSelect.SetSelected(r.Name)
			break
		}
	}

	var orderItems []internal.OrderItem = order.Items
	itemsButton := widget.NewButton("Manage Items", func() {
		products, err := internal.LoadProducts(db)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}
		showOrderItemsDialog(window, products, orderItems, func(items []internal.OrderItem) {
			orderItems = items
		})
	})

	content := container.NewVBox(
		repSelect,
		nameEntry,
		contactEntry,
		dueDatePicker,
		itemsButton,
		commentEntry,
	)

	dialog := dialog.NewCustomConfirm(
		"Edit Order",
		"Save",
		"Cancel",
		content,
		func(submit bool) {
			if !submit {
				return
			}

			dueDate, err := time.Parse("2006-01-02", dueDatePicker.Text)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Invalid due date format. Please use YYYY-MM-DD"), window)
				return
			}

			var repID int64
			for _, r := range representatives {
				if r.Name == repSelect.Selected {
					repID = r.ID
					break
				}
			}

			// Calculate total price
			var totalPrice float64
			for _, item := range orderItems {
				totalPrice += item.Price
			}

			// Update order
			updatedOrder := internal.Order{
				ID:               order.ID,
				DueDate:          dueDate,
				ClientName:       nameEntry.Text,
				Contact:          contactEntry.Text,
				RepresentativeID: repID,
				Comment:          commentEntry.Text,
				TotalPrice:       totalPrice,
				Items:            orderItems,
			}

			if err := internal.EditOrder(db, updatedOrder); err != nil {
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

func showOrderItemsDialog(window fyne.Window, products []internal.Product,
	currentItems []internal.OrderItem, onSave func([]internal.OrderItem)) {

	var itemEntries []struct {
		ProductSelect *widget.Select
		QuantityEntry *widget.Entry
		PriceLabel    *widget.Label
		Container     *fyne.Container
	}

	itemsContainer := container.NewVBox()
	totalLabel := widget.NewLabel("Total: R0.00")

	updateTotalPrice := func() {
		var total float64
		for _, entry := range itemEntries {
			if entry.ProductSelect.Selected != "" {
				quantity, _ := strconv.Atoi(entry.QuantityEntry.Text)
				for _, p := range products {
					if p.Name == entry.ProductSelect.Selected {
						itemTotal := p.Price * float64(quantity)
						total += itemTotal
						entry.PriceLabel.SetText(fmt.Sprintf("Price: R%.2f", itemTotal))
						break
					}
				}
			}
		}
		totalLabel.SetText(fmt.Sprintf("Total: R%.2f", total))
	}

	addItemEntry := func() {
		entry := struct {
			ProductSelect *widget.Select
			QuantityEntry *widget.Entry
			PriceLabel    *widget.Label
			Container     *fyne.Container
		}{
			ProductSelect: widget.NewSelect(nil, nil),
			QuantityEntry: widget.NewEntry(),
			PriceLabel:    widget.NewLabel("Price: R0.00"),
		}

		var productNames []string
		for _, p := range products {
			productNames = append(productNames, p.Name)
		}
		entry.ProductSelect.Options = productNames

		entry.QuantityEntry.SetPlaceHolder("Quantity")

		entry.ProductSelect.OnChanged = func(string) {
			updateTotalPrice()
		}

		entry.QuantityEntry.OnChanged = func(string) {
			updateTotalPrice()
		}

		deleteBtn := widget.NewButton("X", func() {
			index := -1
			for i, e := range itemEntries {
				if e.ProductSelect == entry.ProductSelect {
					index = i
					break
				}
			}
			if index >= 0 {
				itemEntries = append(itemEntries[:index], itemEntries[index+1:]...)
				itemsContainer.Remove(entry.Container)
				updateTotalPrice()
			}
		})

		entry.Container = container.NewHBox(
			container.NewGridWithRows(1,
				entry.ProductSelect,
				entry.QuantityEntry,
				entry.PriceLabel,
				deleteBtn,
			),
		)

		itemEntries = append(itemEntries, entry)
		itemsContainer.Add(entry.Container)
	}

	// Add existing items
	for _, item := range currentItems {
		addItemEntry()
		lastEntry := itemEntries[len(itemEntries)-1]
		for _, p := range products {
			if p.ID == item.ProductID {
				lastEntry.ProductSelect.SetSelected(p.Name)
				break
			}
		}
		lastEntry.QuantityEntry.SetText(fmt.Sprintf("%d", item.Quantity))
	}

	addButton := widget.NewButton("Add Item", addItemEntry)

	saveBtn := widget.NewButton("Save", func() {
		var items []internal.OrderItem
		for _, entry := range itemEntries {
			if entry.ProductSelect.Selected != "" {
				quantity, _ := strconv.Atoi(entry.QuantityEntry.Text)
				var productID int64
				var price float64
				for _, p := range products {
					if p.Name == entry.ProductSelect.Selected {
						productID = p.ID
						price = p.Price * float64(quantity)
						break
					}
				}
				items = append(items, internal.OrderItem{
					ProductID: productID,
					Quantity:  quantity,
					Price:     price,
				})
			}
		}
		onSave(items)
	})

	content := container.NewVBox(
		itemsContainer,
		addButton,
		totalLabel,
		saveBtn,
	)

	dialog := dialog.NewCustom("Order Items", "Close", content, window)
	dialog.Resize(fyne.NewSize(600, 400))
	dialog.Show()
}
