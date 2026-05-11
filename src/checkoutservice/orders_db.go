// Copyright 2026 Akamai
//
// Licensed under the Apache License, Version 2.0 (the "License").

// Persistence of placed orders to a managed PostgreSQL database
// (e.g. Linode Managed PostgreSQL).
//
// Activated only when ORDER_DB_DSN env var is set. Otherwise the
// service falls back to fire-and-forget behaviour, preserving the
// upstream contract.
package main

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"time"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/checkoutservice/genproto"
	_ "github.com/lib/pq"
)

var (
	ordersDB     *sql.DB
	ordersDBOnce sync.Once
)

// initOrdersDB opens a pool against ORDER_DB_DSN and pings it once.
// Safe to call multiple times; idempotent. Returns nil and leaves
// ordersDB == nil when the DSN is absent.
func initOrdersDB() error {
	dsn := os.Getenv("ORDER_DB_DSN")
	if dsn == "" {
		log.Info("ORDER_DB_DSN not set; order persistence disabled.")
		return nil
	}

	var initErr error
	ordersDBOnce.Do(func() {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			initErr = err
			return
		}
		// Conservative pool for a single replica.
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
		log.Info("orders DB ready (PostgreSQL)")
	})
	return initErr
}

// persistOrder writes the placed order and its line items in one
// transaction. Failure does NOT abort the order — we log and return
// so the user-facing PlaceOrder still succeeds.
func persistOrder(ctx context.Context, sessionID, email, currency string,
	order *pb.OrderResult, total *pb.Money) {

	if ordersDB == nil {
		return
	}

	addr := order.GetShippingAddress()
	cost := order.GetShippingCost()
	if cost == nil {
		cost = &pb.Money{}
	}

	tx, err := ordersDB.BeginTx(ctx, nil)
	if err != nil {
		log.Warnf("orders persist: begin tx: %v", err)
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `
INSERT INTO orders (
    order_id, session_id, email, user_currency,
    shipping_tracking_id,
    shipping_cost_currency, shipping_cost_units, shipping_cost_nanos,
    shipping_street, shipping_city, shipping_state, shipping_country, shipping_zip_code,
    total_currency, total_units, total_nanos
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		order.GetOrderId(), sessionID, email, currency,
		order.GetShippingTrackingId(),
		cost.GetCurrencyCode(), cost.GetUnits(), cost.GetNanos(),
		addr.GetStreetAddress(), addr.GetCity(), addr.GetState(),
		addr.GetCountry(), addr.GetZipCode(),
		total.GetCurrencyCode(), total.GetUnits(), total.GetNanos())
	if err != nil {
		log.Warnf("orders persist: insert orders: %v", err)
		return
	}

	for _, it := range order.GetItems() {
		item := it.GetItem()
		ic := it.GetCost()
		if ic == nil {
			ic = &pb.Money{}
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO order_items (
    order_id, product_id, quantity,
    unit_price_currency, unit_price_units, unit_price_nanos
) VALUES ($1,$2,$3,$4,$5,$6)`,
			order.GetOrderId(), item.GetProductId(), item.GetQuantity(),
			ic.GetCurrencyCode(), ic.GetUnits(), ic.GetNanos())
		if err != nil {
			log.Warnf("orders persist: insert order_items (%s): %v",
				item.GetProductId(), err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Warnf("orders persist: commit: %v", err)
		return
	}
	committed = true
	log.Infof("order %s persisted (items=%d)", order.GetOrderId(), len(order.GetItems()))
}
