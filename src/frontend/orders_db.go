// Copyright 2026 Akamai
//
// Licensed under the Apache License, Version 2.0 (the "License").
//
// Read-side access to the orders table on Linode Managed PostgreSQL.
// Activated only when ORDERS_DB_DSN is set; otherwise the frontend
// behaves as before and the /orders pages return an "unavailable"
// notice.
package main

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

var (
	ordersDB     *sql.DB
	ordersDBOnce sync.Once
)

// OrderRow is a flat representation of a row in the orders table,
// suitable for templates.
type OrderRow struct {
	OrderID            string
	SessionID          string
	Email              string
	UserCurrency       string
	ShippingTrackingID string
	ShippingStreet     string
	ShippingCity       string
	ShippingState      string
	ShippingCountry    string
	ShippingZipCode    int32
	TotalCurrency      string
	TotalUnits         int64
	TotalNanos         int32
	CreatedAt          time.Time

	Items []OrderItemRow
}

// OrderItemRow is one row in order_items.
type OrderItemRow struct {
	ProductID         string
	Quantity          int32
	UnitPriceCurrency string
	UnitPriceUnits    int64
	UnitPriceNanos    int32
}

// initOrdersDB opens (once) a connection pool to ORDERS_DB_DSN. If
// the env var is empty, ordersDB stays nil and the read paths just
// degrade to "feature disabled".
func initOrdersDB() error {
	dsn := os.Getenv("ORDERS_DB_DSN")
	if dsn == "" {
		return nil
	}
	var initErr error
	ordersDBOnce.Do(func() {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			initErr = err
			return
		}
		db.SetMaxOpenConns(8)
		db.SetMaxIdleConns(2)
		db.SetConnMaxIdleTime(5 * time.Minute)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			initErr = err
			return
		}
		ordersDB = db
	})
	return initErr
}

// ordersAvailable reports whether the orders DB is configured.
func ordersAvailable() bool { return ordersDB != nil }

// listOrdersBySession returns orders belonging to a session, newest first.
// Limited to `limit` rows (use a small number for UI lists).
func listOrdersBySession(ctx context.Context, sessionID string, limit int) ([]OrderRow, error) {
	if ordersDB == nil {
		return nil, nil
	}
	return queryOrders(ctx, `
SELECT order_id, session_id, COALESCE(email,''), user_currency,
       shipping_tracking_id,
       COALESCE(shipping_street,''), COALESCE(shipping_city,''),
       COALESCE(shipping_state,''), COALESCE(shipping_country,''),
       COALESCE(shipping_zip_code,0),
       COALESCE(total_currency,''), COALESCE(total_units,0), COALESCE(total_nanos,0),
       created_at
FROM orders
WHERE session_id = $1
ORDER BY created_at DESC
LIMIT $2`, sessionID, limit)
}

// listAllOrders returns recent orders across all sessions, newest first.
func listAllOrders(ctx context.Context, limit int) ([]OrderRow, error) {
	if ordersDB == nil {
		return nil, nil
	}
	return queryOrders(ctx, `
SELECT order_id, session_id, COALESCE(email,''), user_currency,
       shipping_tracking_id,
       COALESCE(shipping_street,''), COALESCE(shipping_city,''),
       COALESCE(shipping_state,''), COALESCE(shipping_country,''),
       COALESCE(shipping_zip_code,0),
       COALESCE(total_currency,''), COALESCE(total_units,0), COALESCE(total_nanos,0),
       created_at
FROM orders
ORDER BY created_at DESC
LIMIT $1`, limit)
}

func queryOrders(ctx context.Context, q string, args ...interface{}) ([]OrderRow, error) {
	rows, err := ordersDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []OrderRow
	for rows.Next() {
		var o OrderRow
		if err := rows.Scan(
			&o.OrderID, &o.SessionID, &o.Email, &o.UserCurrency,
			&o.ShippingTrackingID,
			&o.ShippingStreet, &o.ShippingCity, &o.ShippingState,
			&o.ShippingCountry, &o.ShippingZipCode,
			&o.TotalCurrency, &o.TotalUnits, &o.TotalNanos,
			&o.CreatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Attach items for each order (one extra round-trip per order — fine
	// for the small page sizes the UI uses).
	for i := range orders {
		items, err := listOrderItems(ctx, orders[i].OrderID)
		if err != nil {
			return nil, err
		}
		orders[i].Items = items
	}
	return orders, nil
}

func listOrderItems(ctx context.Context, orderID string) ([]OrderItemRow, error) {
	rows, err := ordersDB.QueryContext(ctx, `
SELECT product_id, quantity,
       COALESCE(unit_price_currency,''), COALESCE(unit_price_units,0), COALESCE(unit_price_nanos,0)
FROM order_items
WHERE order_id = $1
ORDER BY id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrderItemRow
	for rows.Next() {
		var it OrderItemRow
		if err := rows.Scan(&it.ProductID, &it.Quantity,
			&it.UnitPriceCurrency, &it.UnitPriceUnits, &it.UnitPriceNanos); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
