package main

import (
	"blissfulBytes-management/internal"
	"blissfulBytes-management/shared/db"
	"fmt"
	"log"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type Order struct {
	ID              int
	CreatedAt       time.Time
	ClientName      string
	Contact         string
	Quantity        int
	Price           float64
	NeedsDelivery   bool
	DeliveryAddress string
	Completed       bool
}

func main() {
	db, err := db.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	myApp := app.New()
	myWindow := myApp.NewWindow("Blissful Bites Order Management")

	// Create input fields
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Client Name")

	contactEntry := widget.NewEntry()
	contactEntry.SetPlaceHolder("Contact (Phone)")

	select_product := widget.NewSelectEntry([]string{"Samoosas Dozen", "Samoosas Single"})
	select_product.SetPlaceHolder("Select Product")

	select_flavor := widget.NewSelectEntry([]string{"Chicken Mince", "Beef Mince", "Cheese & Sweet Corn"})
	select_flavor.SetPlaceHolder("Select Flavor")

	quantityEntry := widget.NewEntry()
	quantityEntry.SetPlaceHolder("Quantity")

	priceEntry := widget.NewEntry()
	priceEntry.SetPlaceHolder("Price")

	deliveryCheck := widget.NewCheck("Needs Delivery", func(bool) {})

	addressEntry := widget.NewMultiLineEntry()
	addressEntry.SetPlaceHolder("Delivery Address")

	commentEntry := widget.NewMultiLineEntry()
	commentEntry.SetPlaceHolder("Comment")

	// Initialize table first
	orderTable := widget.NewTable(
		func() (int, int) { return 6, 4 }, // Initial size
		func() fyne.CanvasObject {
			return widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{})
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {},
	)

	// Define refresh function
	refreshTable := func() {
		orders, err := internal.LoadOrders(db)
		if err != nil {
			log.Printf("Error loading orders: %v", err)
			return
		}

		log.Printf("Refreshing table with %d orders", len(orders))

		orderTable.Length = func() (int, int) {
			return len(orders) + 1, 4 // +1 for header row
		}
		// orderTable.Size().AddWidthHeight(50, 100)

		orderTable.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
			orderTable.SetColumnWidth(0, 150) // Set the Date column width to 200 pixels
			orderTable.SetColumnWidth(1, 200) // Set the Client column width to 200 pixels
			orderTable.SetColumnWidth(2, 80)  // Set the Quantity column width to 200 pixels
			orderTable.SetColumnWidth(3, 100) // Set the Price column width to 200 pixels
			// orderTable.SetColumnWidth(4, 50)  // Set the Status column width to 200 pixels

			label := cell.(*widget.Label)
			if id.Row == 0 {
				// Header row
				switch id.Col {
				case 0:
					label.SetText("Date")
				case 1:
					label.SetText("Client")
				case 2:
					label.SetText("Quantity")
				case 3:
					label.SetText("Price")
					// case 4:
					// 	label.SetText("Status")
				}
				return
			}

			order := orders[id.Row-1]
			switch id.Col {
			case 0:
				label.SetText(order.CreatedAt.Format("2006-01-02 15:04"))
				// label.SetText(order.CreatedAt)
			case 1:
				label.SetText(order.ClientName)
			case 2:
				label.SetText(strconv.Itoa(order.Quantity))
			case 3:
				label.SetText(fmt.Sprintf("R%.2f", order.Price))
				// case 4:
				// 	if order.Completed {
				// 		label.SetText("Completed")
				// 	} else {
				// 		label.SetText("Pending")
				// 	}
			}
		}

		orderTable.Refresh()
	}
	refreshTable()

	// Track selected row
	var selectedRow int = -1

	orderTable.OnSelected = func(id widget.TableCellID) {
		selectedRow = id.Row
	}

	// Submit button handler
	submitBtn := widget.NewButton("Add Order", func() {
		quantity, err := strconv.Atoi(quantityEntry.Text)
		if err != nil {
			log.Printf("Invalid quantity: %v", err)
			return
		}

		price, err := strconv.ParseFloat(priceEntry.Text, 64)
		if err != nil {
			log.Printf("Invalid price: %v", err)
			return
		}

		// Insert into database
		_, err = db.Exec(`
            INSERT INTO orders (
                created_at,
                client_name,
                contact,
                select_product,
                select_flavor,
                quantity,
                price,
                needs_delivery,
                delivery_address,
                comment,
                completed
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			time.Now(),
			nameEntry.Text,
			contactEntry.Text,
			select_product.SelectedText(),
			select_flavor.SelectedText(),
			quantity,
			price,
			deliveryCheck.Checked,
			addressEntry.Text,
			commentEntry.Text,
			false,
		)

		if err != nil {
			log.Printf("Error saving order: %v", err)
			return
		}

		// Clear the form
		nameEntry.Text = ""
		contactEntry.Text = ""
		select_product.Refresh()
		select_flavor.Refresh()
		quantityEntry.Text = ""
		priceEntry.Text = ""
		deliveryCheck.SetChecked(false)
		addressEntry.Text = ""
		commentEntry.Text = ""
		// Refresh the table
		refreshTable()
	})

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

	addOrderBtn := widget.NewButton("+", func() {
		dlg := dialog.NewCustom(
			"New Order",
			"Close",
			container.NewVBox(
				widget.NewLabel("New Order"),
				nameEntry,
				contactEntry,
				select_product,
				select_flavor,
				quantityEntry,
				priceEntry,
				deliveryCheck,
				addressEntry,
				commentEntry,
				submitBtn,
			),
			myWindow,
		)
		dlg.Resize(fyne.NewSize(350, 350))
		dlg.Show()
	})
	// Create the layout
	form := container.NewVBox(
		widget.NewLabel("New Order"),
		addOrderBtn,
		// nameEntry,
		// contactEntry,
		// select_product,
		// select_flavor,
		// quantityEntry,
		// priceEntry,
		// deliveryCheck,
		// addressEntry,
		// commentEntry,
		// submitBtn,
	)
	orderTable.MinSize().Min(fyne.NewSize(500, 500))

	split := container.NewHSplit(
		form,
		container.NewBorder(
			widget.NewLabel("Orders"),
			completeBtn,
			nil,
			nil,
			orderTable,
		),
	)
	split.SetOffset(0.03) // Adjust the split position

	myWindow.SetContent(split)
	myWindow.Resize(fyne.NewSize(1024, 768))

	// Initial table load
	refreshTable()

	menuItem := &fyne.Menu{
		Label: "File",
		Items: nil,
	}
	menu := fyne.NewMainMenu(menuItem)
	myWindow.SetMainMenu(menu)
	myWindow.ShowAndRun()
}
