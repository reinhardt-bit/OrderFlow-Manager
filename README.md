# Blissful Bytes Manager

A desktop application for managing orders, products, and representatives for Blissful Bytes.

## Prerequisites

Before running the application, you need to set up your Turso database:

1. Create a Turso account:
   - Visit [https://app.turso.tech/signup](https://app.turso.tech/signup)
   - Complete the registration process

2. Create a database:
   - Log in to your Turso dashboard
   - Click on "Create Database"
   - Follow the prompts to create your new database

3. Get your database credentials:
   - Navigate to your database in the Turso dashboard
   - Copy your database URL
   - Create a new auth token with read and write permissions
   - Save both the URL and token for use in the application

## Installation

~~### Pre-built Binaries
Pre-built binaries are available for download in the [here](https://github.com/reinhardt-bit/Blissful-Bites-Manager/tree/main/build). Choose the appropriate version for your operating system:
- Windows (64-bit)
- Linux (64-bit)~~

### Running the Application

1. Download and extract the application package for your platform

2. Launch the application

3. On first launch, you'll be prompted to enter your Turso database credentials:
   - Enter your Database URL
   - Enter your Auth Token
   - Click "Save" to continue

## Features

- **Product Management**
  - Add new products
  - Edit existing products
  - Deactivate products
  - Set prices

- **Representative Management**
  - Add new representatives
  - Manage active representatives
  - Deactivate representatives

- **Order Management**
  - Create new orders
  - Track order status
  - Mark orders as completed
  - Export orders to Excel

- **Export Functionality**
  - Export complete order history to Excel
  - Includes all order details
  - Filter and sort capabilities in exported files

## Support

For bug reports and feature requests, please open an issue in the GitHub repository.

## License

[MIT](https://github.com/reinhardt-bit/Order-Tracker?tab=MIT-1-ov-file#readme)
